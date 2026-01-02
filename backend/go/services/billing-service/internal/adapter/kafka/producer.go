// Package kafka provides Kafka producer implementations.
package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/config"
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
	config.Producer.Retry.Max = 3

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

	if _, _, err := p.producer.SendMessage(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// ensureTenantHeaders clones provided headers and injects the tenant header when available.
func ensureTenantHeaders(ctx context.Context, headers []sarama.RecordHeader) []sarama.RecordHeader {
	if tenantID, err := authmw.TenantIDFromContext(ctx); err == nil && tenantID != "" {
		for i := range headers {
			if string(headers[i].Key) == authmw.TenantIDHeader {
				if len(headers[i].Value) == 0 {
					headers[i].Value = []byte(tenantID)
				}
				return headers
			}
		}
		return append(headers, sarama.RecordHeader{Key: []byte(authmw.TenantIDHeader), Value: []byte(tenantID)})
	}

	return headers
}

// Close closes the Kafka producer.
func (p *Producer) Close() error {
	return p.producer.Close()
}
