// Package kafka provides Kafka producer implementations.
package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"

	"tenant-manager/internal/config"
)

// Producer wraps Kafka producer.
type Producer struct {
	producer sarama.SyncProducer
}

// NewProducer creates a new Kafka producer.
func NewProducer(cfg config.KafkaConfig) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = cfg.RetryMax
	if config.Producer.Retry.Max <= 0 {
		config.Producer.Retry.Max = 3
	}
	if cfg.RetryBackoff > 0 {
		config.Producer.Retry.Backoff = cfg.RetryBackoff
	}

	producer, err := sarama.NewSyncProducer(cfg.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	return &Producer{producer: producer}, nil
}

// SendMessage sends a message to a Kafka topic, propagating the tenant header when present.
func (p *Producer) SendMessage(ctx context.Context, topic string, key, value []byte) error {
	headers := ensureTenantHeaders(ctx, nil)

	msg := &sarama.ProducerMessage{
		Topic:   topic,
		Key:     sarama.ByteEncoder(key),
		Value:   sarama.ByteEncoder(value),
		Headers: headers,
	}

	_, _, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// Close closes the Kafka producer.
func (p *Producer) Close() error {
	return p.producer.Close()
}

// ensureTenantHeaders clones the provided headers and injects the tenant header when available.
func ensureTenantHeaders(ctx context.Context, headers []sarama.RecordHeader) []sarama.RecordHeader {
	tenantID, err := authmw.TenantIDFromContext(ctx)
	if err != nil || tenantID == "" {
		return cloneHeaders(headers)
	}

	cloned := cloneHeaders(headers)
	for i := range cloned {
		if string(cloned[i].Key) == authmw.TenantIDHeader {
			if len(cloned[i].Value) == 0 {
				cloned[i].Value = []byte(tenantID)
			}
			return cloned
		}
	}

	return append(cloned, sarama.RecordHeader{Key: []byte(authmw.TenantIDHeader), Value: []byte(tenantID)})
}

// cloneHeaders returns a shallow copy to avoid mutating the caller's slice.
func cloneHeaders(headers []sarama.RecordHeader) []sarama.RecordHeader {
	if len(headers) == 0 {
		return nil
	}

	cloned := make([]sarama.RecordHeader, len(headers))
	copy(cloned, headers)
	return cloned
}
