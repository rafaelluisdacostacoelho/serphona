package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/events"
)

// KafkaPublisher implements EventPublisher using Kafka
type KafkaPublisher struct {
	brokers []string
	topic   string
	// producer kafka.Producer // Will be added when integrating actual Kafka library
}

// NewKafkaPublisher creates a new Kafka event publisher
func NewKafkaPublisher(brokers []string, topic string) (events.EventPublisher, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}

	publisher := &KafkaPublisher{
		brokers: brokers,
		topic:   topic,
	}

	// TODO: Initialize Kafka producer when library is added
	// Example:
	// producer, err := kafka.NewProducer(&kafka.ConfigMap{
	//     "bootstrap.servers": strings.Join(brokers, ","),
	//     "acks": "all",
	// })
	// if err != nil {
	//     return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	// }
	// publisher.producer = producer

	log.Printf("✅ Kafka publisher initialized (brokers: %v, topic: %s)", brokers, topic)
	return publisher, nil
}

// Publish publishes an event synchronously to Kafka
func (p *KafkaPublisher) Publish(ctx context.Context, event *events.Event) error {
	// Marshal event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// TODO: Produce to Kafka when library is added
	// Example:
	// deliveryChan := make(chan kafka.Event)
	// err = p.producer.Produce(&kafka.Message{
	//     TopicPartition: kafka.TopicPartition{Topic: &p.topic, Partition: kafka.PartitionAny},
	//     Key:            []byte(event.TenantID),
	//     Value:          eventJSON,
	// }, deliveryChan)
	// if err != nil {
	//     return fmt.Errorf("failed to produce event: %w", err)
	// }
	//
	// // Wait for delivery confirmation
	// e := <-deliveryChan
	// m := e.(*kafka.Message)
	// if m.TopicPartition.Error != nil {
	//     return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
	// }

	log.Printf("📤 Event published: type=%s, tenant=%s, size=%d bytes", event.Type, event.TenantID, len(eventJSON))
	return nil
}

// PublishAsync publishes an event asynchronously to Kafka (fire and forget)
func (p *KafkaPublisher) PublishAsync(ctx context.Context, event *events.Event) {
	go func() {
		if err := p.Publish(ctx, event); err != nil {
			log.Printf("⚠️  Failed to publish event asynchronously: %v", err)
		}
	}()
}

// Close closes the Kafka producer
func (p *KafkaPublisher) Close() error {
	// TODO: Close Kafka producer when library is added
	// Example:
	// p.producer.Flush(5000) // Wait up to 5 seconds
	// p.producer.Close()

	log.Println("👋 Kafka publisher closed")
	return nil
}
