package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	retry "github.com/avast/retry-go/v4"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/events"
)

// IngestionPublisher publishes rag.ingestion.requested events.
type IngestionPublisher interface {
	PublishIngestion(ctx context.Context, evt events.RAGIngestionRequestedEvent) error
	Close() error
}

type kafkaIngestionWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// KafkaRAGIngestionPublisher sends RAG ingestion events to Kafka.
type KafkaRAGIngestionPublisher struct {
	writer   kafkaIngestionWriter
	backoff  time.Duration
	retryMax uint
	topic    string
}

// NewKafkaRAGIngestionPublisher builds a Kafka publisher for rag.ingestion.requested.
func NewKafkaRAGIngestionPublisher(brokers []string, topic, clientID string, retryMax uint, backoff time.Duration, saslMechanism, saslUsername, saslPassword string) IngestionPublisher {
	if len(brokers) == 0 || topic == "" {
		return NewNoopIngestionPublisher()
	}
	transport := &kafka.Transport{ClientID: clientID}
	if saslMechanism != "" && saslUsername != "" && saslPassword != "" {
		switch saslMechanism {
		case "plain", "PLAIN":
			mech := plain.Mechanism{Username: saslUsername, Password: saslPassword}
			transport.SASL = mech
			dialer := &kafka.Dialer{ClientID: clientID, SASLMechanism: mech}
			transport.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
				conn, err := dialer.DialContext(ctx, network, address)
				if err != nil {
					return nil, err
				}
				return net.Conn(conn), nil
			}
		default:
			log.Printf("rag_ingest_publish unsupported_sasl mechanism=%s", saslMechanism)
		}
	}
	w := &kafka.Writer{
		Addr:      kafka.TCP(brokers...),
		Topic:     topic,
		Balancer:  &kafka.LeastBytes{},
		Async:     false,
		Transport: transport,
	}
	return &KafkaRAGIngestionPublisher{writer: w, backoff: backoff, retryMax: retryMax, topic: topic}
}

// PublishIngestion sends the event.
func (p *KafkaRAGIngestionPublisher) PublishIngestion(ctx context.Context, evt events.RAGIngestionRequestedEvent) error {
	if p == nil || p.writer == nil {
		return nil
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal ingestion event: %w", err)
	}
	return retry.Do(
		func() error {
			err := p.writer.WriteMessages(ctx, kafka.Message{Value: payload})
			if err != nil {
				log.Printf("rag_ingest_publish sink=kafka topic=%s outcome=retryable err=%v", p.topic, err)
				return err
			}
			return nil
		},
		retry.Attempts(p.retryMax),
		retry.Delay(p.backoff),
		retry.Context(ctx),
	)
}

// Close closes the writer if any.
func (p *KafkaRAGIngestionPublisher) Close() error {
	if p != nil && p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// NoopIngestionPublisher drops events.
type NoopIngestionPublisher struct{}

// NewNoopIngestionPublisher returns a no-op publisher.
func NewNoopIngestionPublisher() IngestionPublisher { return &NoopIngestionPublisher{} }

// PublishIngestion drops events.
func (n *NoopIngestionPublisher) PublishIngestion(_ context.Context, _ events.RAGIngestionRequestedEvent) error {
	return nil
}

// Close is noop.
func (n *NoopIngestionPublisher) Close() error { return nil }
