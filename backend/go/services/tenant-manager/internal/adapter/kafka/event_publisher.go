// Package kafka provides Kafka producer implementations.
package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"tenant-manager/internal/domain/tenant"
)

// ProducerSender represents the minimal producer behavior used by the publisher.
type ProducerSender interface {
	SendMessage(ctx context.Context, topic string, key, value []byte) error
}

// EventPublisher implements tenant.EventPublisher using Kafka.
type EventPublisher struct {
	producer    ProducerSender
	topicPrefix string
	dlqTopic    string
}

// NewEventPublisher creates a new Kafka event publisher.
func NewEventPublisher(producer ProducerSender, topicPrefix, dlqTopic string) *EventPublisher {
	return &EventPublisher{
		producer:    producer,
		topicPrefix: topicPrefix,
		dlqTopic:    dlqTopic,
	}
}

// PublishCreated publishes a tenant created event.
func (p *EventPublisher) PublishCreated(ctx context.Context, t *tenant.Tenant) error {
	return p.publishEvent(ctx, "tenant.created", t.ID.String(), t)
}

// PublishUpdated publishes a tenant updated event.
func (p *EventPublisher) PublishUpdated(ctx context.Context, t *tenant.Tenant) error {
	return p.publishEvent(ctx, "tenant.updated", t.ID.String(), t)
}

// PublishDeleted publishes a tenant deleted event.
func (p *EventPublisher) PublishDeleted(ctx context.Context, tenantID uuid.UUID) error {
	event := map[string]string{"tenant_id": tenantID.String()}
	return p.publishEvent(ctx, "tenant.deleted", tenantID.String(), event)
}

// PublishActivated publishes a tenant activated event.
func (p *EventPublisher) PublishActivated(ctx context.Context, t *tenant.Tenant) error {
	return p.publishEvent(ctx, "tenant.activated", t.ID.String(), t)
}

// PublishSuspended publishes a tenant suspended event.
func (p *EventPublisher) PublishSuspended(ctx context.Context, t *tenant.Tenant) error {
	return p.publishEvent(ctx, "tenant.suspended", t.ID.String(), t)
}

// PublishSettingsUpdated publishes a settings updated event.
func (p *EventPublisher) PublishSettingsUpdated(ctx context.Context, tenantID uuid.UUID, settings *tenant.Settings) error {
	event := map[string]any{
		"tenant_id": tenantID.String(),
		"settings":  settings,
	}
	return p.publishEvent(ctx, "tenant.settings.updated", tenantID.String(), event)
}

// PublishUsageReported publishes a usage.reported event with the tenant as key.
func (p *EventPublisher) PublishUsageReported(ctx context.Context, evt tenant.UsageReportedEvent) error {
	return p.publishEvent(ctx, "usage.reported", evt.TenantID.String(), evt)
}

// publishEvent publishes an event to Kafka.
func (p *EventPublisher) publishEvent(ctx context.Context, eventType, key string, payload interface{}) error {
	topic := fmt.Sprintf("%s.%s", p.topicPrefix, eventType)

	value, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	if err := p.producer.SendMessage(ctx, topic, []byte(key), value); err != nil {
		if p.dlqTopic == "" {
			return fmt.Errorf("failed to publish event: %w", err)
		}

		dlqPayload, marshalErr := json.Marshal(map[string]any{
			"event_type": eventType,
			"payload":    json.RawMessage(value),
			"error":      err.Error(),
		})
		if marshalErr != nil {
			return fmt.Errorf("failed to publish event: %w", err)
		}

		if dlqErr := p.producer.SendMessage(ctx, p.dlqTopic, []byte(key), dlqPayload); dlqErr != nil {
			return fmt.Errorf("failed to publish event: %w; dlq error: %v", err, dlqErr)
		}
		return nil
	}

	return nil
}
