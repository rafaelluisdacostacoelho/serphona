// Package kafka provides Kafka producer implementations.
package kafka

import (
	"context"

	"github.com/google/uuid"

	"tenant-manager/internal/domain/tenant"
)

// NoopPublisher implements tenant.EventPublisher doing nothing.
type NoopPublisher struct{}

func NewNoopPublisher() *NoopPublisher { return &NoopPublisher{} }

func (p *NoopPublisher) PublishCreated(ctx context.Context, t *tenant.Tenant) error      { return nil }
func (p *NoopPublisher) PublishUpdated(ctx context.Context, t *tenant.Tenant) error      { return nil }
func (p *NoopPublisher) PublishDeleted(ctx context.Context, id uuid.UUID) error          { return nil }
func (p *NoopPublisher) PublishActivated(ctx context.Context, t *tenant.Tenant) error    { return nil }
func (p *NoopPublisher) PublishSuspended(ctx context.Context, t *tenant.Tenant) error    { return nil }
func (p *NoopPublisher) PublishSettingsUpdated(ctx context.Context, id uuid.UUID, s *tenant.Settings) error {
	return nil
}
