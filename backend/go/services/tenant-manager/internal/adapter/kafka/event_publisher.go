// Package kafka provides Kafka producer implementations.
package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"tenant-manager/internal/domain/tenant"
)

// EventPublisher implements tenant.EventPublisher using Kafka.
type EventPublisher struct {
	producer    *Producer
	topicPrefix string
}

// NewEventPublisher creates a new Kafka event publisher.
func NewEventPublisher(producer *Producer, topicPrefix string) *EventPublisher {
	return &EventPublisher{
		producer:    producer,
		topicPrefix: topicPrefix,
	}
}

// PublishCreated publishes a tenant created event.
func (p *EventPublisher) PublishCreated(ctx context.Context, t *tenant.Tenant) error {
	return p.publishEvent("tenant.created", t.ID.String(), t)
}

// PublishUpdated publishes a tenant updated event.
func (p *EventPublisher) PublishUpdated(ctx context.Context, t *tenant.Tenant) error {
	return p.publishEvent("tenant.updated", t.ID.String(), t)
}

// PublishDeleted publishes a tenant deleted event.
func (p *EventPublisher) PublishDeleted(ctx context.Context, tenantID uuid.UUID) error {
	event := map[string]string{"tenant_id": tenantID.String()}
	return p.publishEvent("tenant.deleted", tenantID.String(), event)
}

// PublishActivated publishes a tenant activated event.
func (p *EventPublisher) PublishActivated(ctx context.Context, t *tenant.Tenant) error {
	return p.publishEvent("tenant.activated", t.ID.String(), t)
}

// PublishSuspended publishes a tenant suspended event.
func (p *EventPublisher) PublishSuspended(ctx context.Context, t *tenant.Tenant) error {
	return p.publishEvent("tenant.suspended", t.ID.String(), t)
}

// PublishSettingsUpdated publishes a settings updated event.
func (p *EventPublisher) PublishSettingsUpdated(ctx context.Context, tenantID uuid.UUID, settings *tenant.Settings) error {
	event := map[string]interface{}{
		"tenant_id": tenantID.String(),
		"settings":  settings,
	}
	return p.publishEvent("tenant.settings.updated", tenantID.String(), event)
}

// publishEvent publishes an event to Kafka.
func (p *EventPublisher) publishEvent(eventType, key string, payload interface{}) error {
	topic := fmt.Sprintf("%s.%s", p.topicPrefix, eventType)

	value, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	if err := p.producer.SendMessage(topic, []byte(key), value); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
