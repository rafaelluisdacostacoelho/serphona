package publisher

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/serphona/serphona/backend/go/libs/platform-events/config"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

// Publisher é responsável por publicar eventos no Kafka
type Publisher struct {
	writer *kafka.Writer
	config *config.Config
	mu     sync.RWMutex
	closed bool
}

// New cria um novo publisher
func New(cfg *config.Config) (*Publisher, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    cfg.PublisherBatchSize,
		BatchTimeout: cfg.PublisherBatchTimeout,
		MaxAttempts:  cfg.PublisherMaxRetries,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		Compression:  kafka.Snappy,
	}

	p := &Publisher{
		writer: writer,
		config: cfg,
	}

	if cfg.Debug {
		log.Printf("[platform-events] Publisher initialized with brokers: %v", cfg.Brokers)
	}

	return p, nil
}

// Publish publica um evento em um tópico específico
func (p *Publisher) Publish(ctx context.Context, topic string, event *types.Event) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPublisherClosed
	}

	// Serializar evento
	data, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Criar mensagem Kafka
	msg := kafka.Message{
		Topic: topic,
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

	if event.TraceID != "" {
		msg.Headers = append(msg.Headers, kafka.Header{
			Key:   "trace_id",
			Value: []byte(event.TraceID),
		})
	}

	// Publicar no Kafka
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
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
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPublisherClosed
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
			Topic: topic,
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

		if event.TraceID != "" {
			msg.Headers = append(msg.Headers, kafka.Header{
				Key:   "trace_id",
				Value: []byte(event.TraceID),
			})
		}

		messages = append(messages, msg)
	}

	// Publicar batch no Kafka
	err := p.writer.WriteMessages(ctx, messages...)
	if err != nil {
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

	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("failed to close publisher: %w", err)
	}

	if p.config.Debug {
		log.Println("[platform-events] Publisher closed")
	}

	return nil
}

// Stats retorna estatísticas do publisher
func (p *Publisher) Stats() kafka.WriterStats {
	return p.writer.Stats()
}

// Errors
var (
	ErrPublisherClosed = fmt.Errorf("publisher is closed")
)
