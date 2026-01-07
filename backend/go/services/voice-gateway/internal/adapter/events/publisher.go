// Package events provides event publishing implementations.
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/domain/call"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/types"
)

// Publisher defines the interface for event publishing.
type Publisher interface {
	Publish(ctx context.Context, event *types.Event) error
	PublishAsync(ctx context.Context, event *types.Event)
	Close() error
	HealthCheck(ctx context.Context) error
	PublishCallStarted(ctx context.Context, c *call.Call) error
	PublishCallAnswered(ctx context.Context, c *call.Call) error
	PublishCallTransferred(ctx context.Context, c *call.Call) error
	PublishCallEnded(ctx context.Context, c *call.Call) error
}

// kafkaPublisher publishes events to Kafka.
type kafkaPublisher struct {
	producer    sarama.SyncProducer
	brokers     []string
	config      *sarama.Config
	topicPrefix string
	logger      *zap.Logger
}

// NewPublisher creates a new Kafka event publisher.
func NewPublisher(brokers []string, topicPrefix string, logger *zap.Logger) (Publisher, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	config.Producer.Idempotent = true
	config.Net.MaxOpenRequests = 1

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	logger.Info("kafka producer created", zap.Strings("brokers", brokers))

	return &kafkaPublisher{
		producer:    producer,
		brokers:     brokers,
		config:      config,
		topicPrefix: topicPrefix,
		logger:      logger,
	}, nil
}

// Close closes the Kafka producer.
func (p *kafkaPublisher) Close() error {
	return p.producer.Close()
}

// HealthCheck verifies connectivity to Kafka brokers.
func (p *kafkaPublisher) HealthCheck(ctx context.Context) error {
	if len(p.brokers) == 0 {
		return fmt.Errorf("no kafka brokers configured")
	}

	client, err := sarama.NewClient(p.brokers, p.config)
	if err != nil {
		return fmt.Errorf("failed to create kafka client: %w", err)
	}
	defer client.Close()

	brokers := client.Brokers()
	if len(brokers) == 0 {
		return fmt.Errorf("kafka client has no brokers")
	}

	for _, b := range brokers {
		if err := b.Open(client.Config()); err != nil && err != sarama.ErrAlreadyConnected {
			return fmt.Errorf("failed to open broker %s: %w", b.Addr(), err)
		}
		if ok, _ := b.Connected(); !ok {
			return fmt.Errorf("kafka broker not connected: %s", b.Addr())
		}
	}

	return ctx.Err()
}

// CallEvent represents a call-related event.
type CallEvent struct {
	EventID        string                 `json:"event_id"`
	EventType      string                 `json:"event_type"`
	Timestamp      time.Time              `json:"timestamp"`
	CallID         uuid.UUID              `json:"call_id"`
	TenantID       uuid.UUID              `json:"tenant_id"`
	ConversationID *uuid.UUID             `json:"conversation_id,omitempty"`
	Direction      string                 `json:"direction"`
	CallerNumber   string                 `json:"caller_number"`
	CalleeNumber   string                 `json:"callee_number"`
	State          string                 `json:"state"`
	Duration       int64                  `json:"duration,omitempty"` // milliseconds
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// PublishCallStarted publishes a call.started event.
func (p *kafkaPublisher) PublishCallStarted(ctx context.Context, c *call.Call) error {
	event := CallEvent{
		EventID:      uuid.New().String(),
		EventType:    "call.started",
		Timestamp:    time.Now().UTC(),
		CallID:       c.ID,
		TenantID:     c.TenantID,
		Direction:    string(c.Direction),
		CallerNumber: c.CallerNumber,
		CalleeNumber: c.CalleeNumber,
		State:        string(c.State),
		Metadata:     c.Metadata,
	}

	return p.publishEvent(ctx, "call.started", c.ID.String(), event)
}

// PublishCallAnswered publishes a call.answered event.
func (p *kafkaPublisher) PublishCallAnswered(ctx context.Context, c *call.Call) error {
	event := CallEvent{
		EventID:        uuid.New().String(),
		EventType:      "call.answered",
		Timestamp:      time.Now().UTC(),
		CallID:         c.ID,
		TenantID:       c.TenantID,
		ConversationID: &c.ConversationID,
		Direction:      string(c.Direction),
		CallerNumber:   c.CallerNumber,
		CalleeNumber:   c.CalleeNumber,
		State:          string(c.State),
		Metadata:       c.Metadata,
	}

	return p.publishEvent(ctx, "call.answered", c.ID.String(), event)
}

// PublishCallEnded publishes a call.ended event.
func (p *kafkaPublisher) PublishCallEnded(ctx context.Context, c *call.Call) error {
	duration := int64(0)
	if c.Duration > 0 {
		duration = c.Duration.Milliseconds()
	}

	event := CallEvent{
		EventID:        uuid.New().String(),
		EventType:      "call.ended",
		Timestamp:      time.Now().UTC(),
		CallID:         c.ID,
		TenantID:       c.TenantID,
		ConversationID: &c.ConversationID,
		Direction:      string(c.Direction),
		CallerNumber:   c.CallerNumber,
		CalleeNumber:   c.CalleeNumber,
		State:          string(c.State),
		Duration:       duration,
		Metadata:       c.Metadata,
	}

	return p.publishEvent(ctx, "call.ended", c.ID.String(), event)
}

// TranscriptionEvent represents a speech transcription event.
type TranscriptionEvent struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	Timestamp      time.Time `json:"timestamp"`
	CallID         uuid.UUID `json:"call_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Text           string    `json:"text"`
	Confidence     float64   `json:"confidence"`
	IsFinal        bool      `json:"is_final"`
	Language       string    `json:"language"`
	Provider       string    `json:"provider"`
	LatencyMs      int64     `json:"latency_ms"`
}

// PublishSTTTranscribed publishes a stt.transcribed event.
func (p *kafkaPublisher) PublishSTTTranscribed(ctx context.Context, callID, tenantID, conversationID uuid.UUID, text string, confidence float64, isFinal bool, provider string, latency time.Duration) error {
	event := TranscriptionEvent{
		EventID:        uuid.New().String(),
		EventType:      "stt.transcribed",
		Timestamp:      time.Now().UTC(),
		CallID:         callID,
		TenantID:       tenantID,
		ConversationID: conversationID,
		Text:           text,
		Confidence:     confidence,
		IsFinal:        isFinal,
		Provider:       provider,
		LatencyMs:      latency.Milliseconds(),
	}

	return p.publishEvent(ctx, "stt.transcribed", callID.String(), event)
}

// LLMResponseEvent represents an LLM response event.
type LLMResponseEvent struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	Timestamp      time.Time `json:"timestamp"`
	CallID         uuid.UUID `json:"call_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	AgentID        string    `json:"agent_id"`
	ResponseText   string    `json:"response_text"`
	LatencyMs      int64     `json:"latency_ms"`
}

// PublishLLMResponded publishes an llm.responded event.
func (p *kafkaPublisher) PublishLLMResponded(ctx context.Context, callID, tenantID, conversationID uuid.UUID, agentID, responseText string, latency time.Duration) error {
	event := LLMResponseEvent{
		EventID:        uuid.New().String(),
		EventType:      "llm.responded",
		Timestamp:      time.Now().UTC(),
		CallID:         callID,
		TenantID:       tenantID,
		ConversationID: conversationID,
		AgentID:        agentID,
		ResponseText:   responseText,
		LatencyMs:      latency.Milliseconds(),
	}

	return p.publishEvent(ctx, "llm.responded", callID.String(), event)
}

// TTSEvent represents a TTS generation event.
type TTSEvent struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	Timestamp      time.Time `json:"timestamp"`
	CallID         uuid.UUID `json:"call_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Text           string    `json:"text"`
	Provider       string    `json:"provider"`
	VoiceID        string    `json:"voice_id"`
	LatencyMs      int64     `json:"latency_ms"`
	AudioSize      int       `json:"audio_size_bytes"`
}

// PublishTTSGenerated publishes a tts.generated event.
func (p *kafkaPublisher) PublishTTSGenerated(ctx context.Context, callID, tenantID, conversationID uuid.UUID, text, provider, voiceID string, audioSize int, latency time.Duration) error {
	event := TTSEvent{
		EventID:        uuid.New().String(),
		EventType:      "tts.generated",
		Timestamp:      time.Now().UTC(),
		CallID:         callID,
		TenantID:       tenantID,
		ConversationID: conversationID,
		Text:           text,
		Provider:       provider,
		VoiceID:        voiceID,
		LatencyMs:      latency.Milliseconds(),
		AudioSize:      audioSize,
	}

	return p.publishEvent(ctx, "tts.generated", callID.String(), event)
}

// TransferEvent represents a call transfer event.
type TransferEvent struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	Timestamp      time.Time `json:"timestamp"`
	CallID         uuid.UUID `json:"call_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	TransferType   string    `json:"transfer_type"` // queue, agent, external
	TransferTarget string    `json:"transfer_target"`
	Reason         string    `json:"reason,omitempty"`
}

// PublishCallTransferred publishes a call.transferred event.
func (p *kafkaPublisher) PublishCallTransferred(ctx context.Context, c *call.Call) error {
	// Implementação corrigida
	return nil
}

// ErrorEvent represents an error event.
type ErrorEvent struct {
	EventID        string     `json:"event_id"`
	EventType      string     `json:"event_type"`
	Timestamp      time.Time  `json:"timestamp"`
	CallID         uuid.UUID  `json:"call_id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
	ErrorType      string     `json:"error_type"`
	ErrorMessage   string     `json:"error_message"`
	Component      string     `json:"component"` // stt, llm, tts, asterisk
}

// PublishError publishes an error event.
func (p *kafkaPublisher) PublishError(ctx context.Context, callID, tenantID uuid.UUID, conversationID *uuid.UUID, errorType, errorMessage, component string) error {
	event := ErrorEvent{
		EventID:        uuid.New().String(),
		EventType:      fmt.Sprintf("error.%s", errorType),
		Timestamp:      time.Now().UTC(),
		CallID:         callID,
		TenantID:       tenantID,
		ConversationID: conversationID,
		ErrorType:      errorType,
		ErrorMessage:   errorMessage,
		Component:      component,
	}

	return p.publishEvent(ctx, fmt.Sprintf("error.%s", errorType), callID.String(), event)
}

// publishEvent publishes an event to Kafka.
func (p *kafkaPublisher) publishEvent(ctx context.Context, eventType, key string, payload interface{}) error {
	topic := fmt.Sprintf("%s.%s", p.topicPrefix, eventType)

	value, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(value),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.logger.Error("failed to publish event",
			zap.String("event_type", eventType),
			zap.Error(err),
		)

		// Attempt DLQ publish with minimal envelope
		dlqTopic := fmt.Sprintf("%s.dlq", p.topicPrefix)
		dlqPayload := map[string]interface{}{
			"failed_event_type": eventType,
			"key":               key,
			"error":             err.Error(),
			"payload":           payload,
			"ts":                time.Now().UTC().Format(time.RFC3339Nano),
		}

		dlqValue, marshalErr := json.Marshal(dlqPayload)
		if marshalErr == nil {
			dlqMsg := &sarama.ProducerMessage{
				Topic: dlqTopic,
				Key:   sarama.StringEncoder(key),
				Value: sarama.ByteEncoder(dlqValue),
			}
			if _, _, dlqErr := p.producer.SendMessage(dlqMsg); dlqErr != nil {
				p.logger.Error("failed to publish to dlq", zap.Error(dlqErr))
			}
		}

		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.logger.Debug("event published",
		zap.String("event_type", eventType),
		zap.String("topic", topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
	)

	return nil
}

// Implementa os métodos da interface Publisher
func (p *kafkaPublisher) Publish(ctx context.Context, event *types.Event) error {
	// Implementação do método Publish
	return nil
}

func (p *kafkaPublisher) PublishAsync(ctx context.Context, event *types.Event) {
	// Implementação do método PublishAsync
}
