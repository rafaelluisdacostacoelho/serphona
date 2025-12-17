package publisher

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/serphona/serphona/backend/go/libs/platform-events/config"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

// Publisher é responsável por publicar eventos no Kafka
type Publisher struct {
	writer *kafka.Writer
	ready  map[string]bool
	config *config.Config
	mu     sync.RWMutex
	closed bool
}

// New cria um novo publisher
func New(cfg *config.Config) (*Publisher, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	p := &Publisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(cfg.Brokers...),
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
			BatchSize:              cfg.PublisherBatchSize,
			BatchTimeout:           cfg.PublisherBatchTimeout,
			MaxAttempts:            cfg.PublisherMaxRetries,
			ReadTimeout:            10 * time.Second,
			WriteTimeout:           10 * time.Second,
			RequiredAcks:           kafka.RequireOne,
			Async:                  false,
			Compression:            kafka.Snappy,
			WriteBackoffMin:        cfg.PublisherRetryInterval,
			WriteBackoffMax:        cfg.PublisherRetryInterval,
			Transport:              newTransport(cfg),
		},
		ready:  make(map[string]bool),
		config: cfg,
	}

	if cfg.Debug {
		log.Printf("[platform-events] Publisher initialized with brokers: %v", cfg.Brokers)
	}

	return p, nil
}

// Publish publica um evento em um tópico específico
func (p *Publisher) Publish(ctx context.Context, topic string, event *types.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPublisherClosed
	}

	if err := p.ensureTopic(ctx, topic); err != nil {
		return err
	}

	// Serializar evento
	data, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Criar mensagem Kafka
	msg := kafka.Message{
		Key:   []byte(event.ID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.Type)},
			{Key: "source", Value: []byte(event.Source)},
			{Key: "version", Value: []byte(event.Version)},
		},
		Time: event.Timestamp,
	}

	// Adicionar headers opcionais
	if event.TenantID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "tenant_id",
			Value: []byte(event.TenantID),
		})
	}

	if event.UserID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "user_id",
			Value: []byte(event.UserID),
		})
	}

	if event.TraceID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "trace_id",
			Value: []byte(event.TraceID),
		})
	}

	if event.SpanID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "span_id",
			Value: []byte(event.SpanID),
		})
	}

	msg.Topic = topic

	conn, err := kafka.DialLeader(ctx, "tcp", p.config.Brokers[0], topic, 0)
	if err != nil {
		return fmt.Errorf("failed to dial leader for topic %s: %w", topic, err)
	}
	defer conn.Close()

	if _, err := conn.WriteMessages(msg); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	if p.config.Debug {
		log.Printf("[platform-events] Published event: topic=%s, type=%s, id=%s",
			topic, event.Type, event.ID)
	}

	return nil
}

// PublishBatch publica múltiplos eventos em batch
func (p *Publisher) PublishBatch(ctx context.Context, topic string, events []*types.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPublisherClosed
	}

	if err := p.ensureTopic(ctx, topic); err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	// Criar mensagens Kafka
	messages := make([]kafka.Message, 0, len(events))
	for _, event := range events {
		data, err := event.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize event %s: %w", event.ID, err)
		}

		msg := kafka.Message{
			Key:   []byte(event.ID),
			Value: data,
			Headers: []kafka.Header{
				{Key: "event_type", Value: []byte(event.Type)},
				{Key: "source", Value: []byte(event.Source)},
				{Key: "version", Value: []byte(event.Version)},
			},
			Time: event.Timestamp,
		}

		if event.TenantID != "" {
			msg.Headers = append(msg.Headers, kafka.Header{
				Key:   "tenant_id",
				Value: []byte(event.TenantID),
			})
		}

		if event.UserID != "" {
			msg.Headers = append(msg.Headers, kafka.Header{
				Key:   "user_id",
				Value: []byte(event.UserID),
			})
		}

		if event.TraceID != "" {
			msg.Headers = append(msg.Headers, kafka.Header{
				Key:   "trace_id",
				Value: []byte(event.TraceID),
			})
		}

		if event.SpanID != "" {
			msg.Headers = append(msg.Headers, kafka.Header{
				Key:   "span_id",
				Value: []byte(event.SpanID),
			})
		}

		msg.Topic = topic
		messages = append(messages, msg)
	}

	conn, err := kafka.DialLeader(ctx, "tcp", p.config.Brokers[0], topic, 0)
	if err != nil {
		return fmt.Errorf("failed to dial leader for topic %s: %w", topic, err)
	}
	defer conn.Close()

	if _, err := conn.WriteMessages(messages...); err != nil {
		return fmt.Errorf("failed to publish batch: %w", err)
	}

	if p.config.Debug {
		log.Printf("[platform-events] Published batch: topic=%s, count=%d", topic, len(events))
	}

	return nil
}

// Close fecha o publisher
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	if p.config.Debug {
		log.Println("[platform-events] Publisher closed")
	}

	return nil
}

// Stats retorna estatísticas do publisher
func (p *Publisher) Stats() kafka.WriterStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return kafka.WriterStats{}
}

// ensureTopic guarantees the topic exists and has a leader before publishing.
func (p *Publisher) ensureTopic(ctx context.Context, topic string) error {
	if p.ready[topic] {
		return nil
	}

	if len(p.config.Brokers) == 0 {
		return fmt.Errorf("no brokers configured")
	}

	conn, err := kafka.DialContext(ctx, "tcp", p.config.Brokers[0])
	if err != nil {
		return fmt.Errorf("failed to dial broker %s: %w", p.config.Brokers[0], err)
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if ok {
		_ = conn.SetDeadline(deadline)
	}

	if err := conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("failed to create topic %s: %w", topic, err)
	}

	parts, err := conn.ReadPartitions(topic)
	if err != nil {
		return fmt.Errorf("failed to read partitions for topic %s: %w", topic, err)
	}
	if len(parts) == 0 {
		return fmt.Errorf("no partitions available for topic %s", topic)
	}

	// Dial leader to ensure metadata is propagated.
	leader, err := kafka.DialLeader(ctx, "tcp", p.config.Brokers[0], topic, 0)
	if err != nil {
		return fmt.Errorf("failed to reach leader for topic %s: %w", topic, err)
	}
	leader.Close()

	p.ready[topic] = true
	return nil
}

// Errors
var (
	ErrPublisherClosed = fmt.Errorf("publisher is closed")
)

func newDialer(cfg *config.Config) (*kafka.Dialer, error) {
	dialer := &kafka.Dialer{
		ClientID: cfg.ClientID,
		Timeout:  10 * time.Second,
	}

	if cfg.UseTLS {
		dialer.TLS = &tls.Config{
			InsecureSkipVerify: cfg.TLSInsecureSkipVerify,
		}
	}

	if cfg.SASLMechanism != "" {
		switch cfg.SASLMechanism {
		case "plain":
			dialer.SASLMechanism = plain.Mechanism{
				Username: cfg.SASLUsername,
				Password: cfg.SASLPassword,
			}
		case "scram-sha256":
			mech, err := scram.Mechanism(scram.SHA256, cfg.SASLUsername, cfg.SASLPassword)
			if err != nil {
				return nil, err
			}
			dialer.SASLMechanism = mech
		case "scram-sha512":
			mech, err := scram.Mechanism(scram.SHA512, cfg.SASLUsername, cfg.SASLPassword)
			if err != nil {
				return nil, err
			}
			dialer.SASLMechanism = mech
		}
	}

	return dialer, nil
}

// newTransport builds a kafka.Transport using the same security settings as the dialer.
func newTransport(cfg *config.Config) *kafka.Transport {
	dialer, err := newDialer(cfg)
	if err != nil {
		// Fallback to default transport if config is invalid; writer creation will surface validate errors earlier.
		return &kafka.Transport{}
	}

	return &kafka.Transport{
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
		SASL: dialer.SASLMechanism,
		TLS:  dialer.TLS,
	}
}
