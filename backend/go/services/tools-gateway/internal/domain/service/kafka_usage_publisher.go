package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	retry "github.com/avast/retry-go/v4"
	"github.com/segmentio/kafka-go"
)

// kafkaWriter defines the subset of kafka.Writer we need for testing.
type kafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// KafkaUsagePublisher publishes usage events to Kafka.
type KafkaUsagePublisher struct {
	writer   kafkaWriter
	backoff  time.Duration
	retryMax uint
	topic    string
}

// NewKafkaUsagePublisher builds a Kafka publisher. If topic or brokers are missing, returns noop.
func NewKafkaUsagePublisher(brokers []string, topic, clientID string, retryMax uint, backoff time.Duration) UsagePublisher {
	if len(brokers) == 0 || topic == "" {
		return NewNoopUsagePublisher()
	}
	initUsageMetrics()

	w := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
		Async:    false,
		Transport: &kafka.Transport{
			ClientID: clientID,
		},
	}

	return &KafkaUsagePublisher{
		writer:   w,
		backoff:  backoff,
		retryMax: retryMax,
		topic:    topic,
	}
}

// PublishUsage sends the usage event to Kafka.
func (p *KafkaUsagePublisher) PublishUsage(ctx context.Context, evt UsageEvent) error {
	if p == nil || p.writer == nil {
		return nil
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal usage event: %w", err)
	}

	return retry.Do(
		func() error {
			usagePublishAttempts.WithLabelValues("kafka", p.topic, "attempt", "").Inc()
			err := p.writer.WriteMessages(ctx, kafka.Message{Value: payload})
			if err != nil {
				usagePublishAttempts.WithLabelValues("kafka", p.topic, "retryable", "").Inc()
				log.Printf("usage_publish sink=kafka outcome=retryable topic=%s err=%v", p.topic, err)
				return err
			}
			usagePublishAttempts.WithLabelValues("kafka", p.topic, "success", "").Inc()
			return nil
		},
		retry.Attempts(p.retryMax),
		retry.Delay(p.backoff),
		retry.Context(ctx),
	)
}

// Close closes the writer if present.
func (p *KafkaUsagePublisher) Close() error {
	if p != nil && p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
