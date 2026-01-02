package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	walletapp "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/application/wallet"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/config"
)

// UsageConsumer wraps a Kafka consumer group to handle usage.reported events.
type UsageConsumer struct {
	group    sarama.ConsumerGroup
	producer sarama.SyncProducer
	cfg      config.KafkaConfig
	wallet   walletDebitService
	logger   *log.Logger
}

type walletDebitService interface {
	DebitUsage(ctx context.Context, evt walletapp.UsageReported) error
}

// NewUsageConsumer creates a consumer group client.
func NewUsageConsumer(cfg config.KafkaConfig, walletSvc walletDebitService, logger *log.Logger) (*UsageConsumer, error) {
	scfg := sarama.NewConfig()
	scfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	scfg.Consumer.Offsets.Initial = sarama.OffsetNewest

	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, scfg)
	if err != nil {
		return nil, fmt.Errorf("create consumer group: %w", err)
	}

	pcfg := sarama.NewConfig()
	pcfg.Producer.Return.Successes = true
	pcfg.Producer.RequiredAcks = sarama.WaitForLocal

	producer, err := sarama.NewSyncProducer(cfg.Brokers, pcfg)
	if err != nil {
		return nil, fmt.Errorf("create producer: %w", err)
	}

	return &UsageConsumer{group: group, producer: producer, cfg: cfg, wallet: walletSvc, logger: logger}, nil
}

// Start begins consuming usage.reported messages until ctx is cancelled.
func (c *UsageConsumer) Start(ctx context.Context) {
	handler := &usageHandler{
		wallet:        c.wallet,
		logger:        c.logger,
		producer:      c.producer,
		retryAttempts: c.cfg.MaxRetries,
		retryBackoff:  time.Duration(c.cfg.RetryBackoffMs) * time.Millisecond,
		dlqTopic:      c.cfg.DLQTopic,
	}
	topics := []string{"usage.reported"}
	if len(c.cfg.Topics) > 0 {
		topics = append(topics, c.cfg.Topics...)
	}

	go func() {
		for {
			if err := c.group.Consume(ctx, topics, handler); err != nil {
				c.logger.Printf("kafka consume error: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()
}

// Close closes the consumer group.
func (c *UsageConsumer) Close() error {
	_ = c.producer.Close()
	return c.group.Close()
}

// usageHandler processes messages and debits wallets.
type usageHandler struct {
	wallet        walletDebitService
	logger        *log.Logger
	producer      sarama.SyncProducer
	retryAttempts int
	retryBackoff  time.Duration
	dlqTopic      string
}

var (
	metricUsageMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "billing_usage_messages_total",
		Help: "Count of usage messages processed by billing-service",
	}, []string{"status"})

	metricUsageLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "billing_usage_process_seconds",
		Help:    "Latency to process a usage message",
		Buckets: prometheus.DefBuckets,
	})

	metricUsageLag = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "billing_usage_kafka_lag",
		Help:    "Kafka lag (messages) observed when consuming usage messages",
		Buckets: []float64{0, 1, 5, 10, 50, 100, 500, 1_000, 5_000, 10_000},
	})
)

func (h *usageHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *usageHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *usageHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		start := time.Now()
		status, err := h.processMessage(session.Context(), msg)
		lag := float64(0)
		high := claim.HighWaterMarkOffset()
		if high > msg.Offset {
			lag = float64(high - msg.Offset - 1)
			metricUsageLag.Observe(lag)
		}

		metricUsageLatency.Observe(time.Since(start).Seconds())
		metricUsageMessages.WithLabelValues(status).Inc()

		if err != nil && status == "error" {
			err = h.retryWithBackoff(session.Context(), msg, err)
		}

		if err != nil && status == "error" {
			if dlqErr := h.sendToDLQ(msg, err, status); dlqErr != nil {
				h.logger.Printf("usage_consume status=dlq_failed tenant=%s request_id=%s trace_id=%s offset=%d lag=%.0f err=%v", extractTenant(msg.Value), extractRequestID(msg.Value), extractTraceID(msg.Value), msg.Offset, lag, dlqErr)
			} else {
				h.logger.Printf("usage_consume status=dlq tenant=%s request_id=%s trace_id=%s offset=%d lag=%.0f err=%v", extractTenant(msg.Value), extractRequestID(msg.Value), extractTraceID(msg.Value), msg.Offset, lag, err)
			}
			continue
		}

		if err != nil {
			h.logger.Printf("usage_consume status=%s tenant=%s request_id=%s trace_id=%s offset=%d lag=%.0f err=%v", status, extractTenant(msg.Value), extractRequestID(msg.Value), extractTraceID(msg.Value), msg.Offset, lag, err)
			continue
		}

		h.logger.Printf("usage_consume status=%s tenant=%s request_id=%s trace_id=%s offset=%d lag=%.0f", status, extractTenant(msg.Value), extractRequestID(msg.Value), extractTraceID(msg.Value), msg.Offset, lag)
		session.MarkMessage(msg, "")
	}
	return nil
}

func (h *usageHandler) processMessage(ctx context.Context, msg *sarama.ConsumerMessage) (string, error) {
	value := msg.Value
	var evt walletapp.UsageReported
	if err := json.Unmarshal(value, &evt); err != nil {
		return "decode_error", fmt.Errorf("decode usage event: %w", err)
	}
	if evt.TenantID == uuid.Nil {
		return "invalid", fmt.Errorf("missing tenant_id")
	}
	if evt.RequestID == "" {
		return "invalid", fmt.Errorf("missing request_id")
	}
	if err := h.wallet.DebitUsage(ctx, evt); err != nil {
		if errors.Is(err, walletapp.ErrDuplicateRequest) {
			return "duplicate", nil
		}
		return "error", fmt.Errorf("debit wallet: %w", err)
	}
	return "success", nil
}

func (h *usageHandler) retryWithBackoff(ctx context.Context, msg *sarama.ConsumerMessage, origErr error) error {
	if h.retryAttempts <= 0 {
		return origErr
	}

	status := "error"
	err := origErr
	for i := 0; i < h.retryAttempts; i++ {
		time.Sleep(h.retryBackoff)
		status, err = h.processMessage(ctx, msg)
		if err == nil {
			return nil
		}
		metricUsageMessages.WithLabelValues(status).Inc()
	}
	return err
}

func (h *usageHandler) sendToDLQ(msg *sarama.ConsumerMessage, lastErr error, status string) error {
	if h.producer == nil || h.dlqTopic == "" {
		return lastErr
	}

	dlqPayload := struct {
		OriginalTopic string          `json:"original_topic"`
		Partition     int32           `json:"partition"`
		Offset        int64           `json:"offset"`
		Error         string          `json:"error"`
		Status        string          `json:"status"`
		Payload       json.RawMessage `json:"payload"`
		TenantID      string          `json:"tenant_id"`
		RequestID     string          `json:"request_id"`
		TraceID       string          `json:"trace_id"`
	}{
		OriginalTopic: msg.Topic,
		Partition:     msg.Partition,
		Offset:        msg.Offset,
		Error:         lastErr.Error(),
		Status:        status,
		Payload:       msg.Value,
		TenantID:      extractTenant(msg.Value),
		RequestID:     extractRequestID(msg.Value),
		TraceID:       extractTraceID(msg.Value),
	}

	encoded, err := json.Marshal(dlqPayload)
	if err != nil {
		return err
	}

	tenantCtx := context.Background()
	if dlqPayload.TenantID != "" {
		tenantCtx = authmw.WithTenantID(tenantCtx, dlqPayload.TenantID)
	}
	headers := ensureTenantHeaders(tenantCtx, nil)

	producerMsg := &sarama.ProducerMessage{
		Topic:   h.dlqTopic,
		Key:     sarama.StringEncoder(dlqPayload.TenantID),
		Value:   sarama.ByteEncoder(encoded),
		Headers: headers,
	}

	_, _, err = h.producer.SendMessage(producerMsg)
	return err
}

func extractTenant(raw []byte) string {
	var aux struct {
		TenantID uuid.UUID `json:"tenant_id"`
	}
	if err := json.Unmarshal(raw, &aux); err != nil {
		return ""
	}
	return aux.TenantID.String()
}

func extractRequestID(raw []byte) string {
	var aux struct {
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(raw, &aux); err != nil {
		return ""
	}
	return aux.RequestID
}

func extractTraceID(raw []byte) string {
	var aux struct {
		TraceID string `json:"trace_id"`
	}
	if err := json.Unmarshal(raw, &aux); err != nil {
		return ""
	}
	return aux.TraceID
}
