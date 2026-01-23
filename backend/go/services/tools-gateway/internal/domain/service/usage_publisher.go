package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UsageEvent represents a billing/usage record for a tool execution.
type UsageEvent struct {
	TenantID uuid.UUID
	ToolID   uuid.UUID
	UserID   uuid.UUID
	Credits  int
	Status   string
	Latency  int64
	At       time.Time
}

// UsagePublisher publishes usage events to downstream billing/analytics sinks.
type UsagePublisher interface {
	PublishUsage(ctx context.Context, evt UsageEvent) error
}

// noopUsagePublisher is used when no publisher is configured.
type noopUsagePublisher struct{}

func (n noopUsagePublisher) PublishUsage(_ context.Context, _ UsageEvent) error {
	return nil
}

// NewNoopUsagePublisher returns a UsagePublisher that discards events.
func NewNoopUsagePublisher() UsagePublisher {
	return noopUsagePublisher{}
}
