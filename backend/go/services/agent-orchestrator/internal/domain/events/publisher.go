package events

import "context"

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	// Publish publishes an event to the event bus
	Publish(ctx context.Context, event *Event) error

	// PublishAsync publishes an event asynchronously (fire and forget)
	PublishAsync(ctx context.Context, event *Event)

	// Close closes the publisher and releases resources
	Close() error
}

// NoOpPublisher is a no-op implementation of EventPublisher
type NoOpPublisher struct{}

// NewNoOpPublisher creates a new no-op publisher
func NewNoOpPublisher() EventPublisher {
	return &NoOpPublisher{}
}

// Publish does nothing (no-op)
func (p *NoOpPublisher) Publish(ctx context.Context, event *Event) error {
	return nil
}

// PublishAsync does nothing (no-op)
func (p *NoOpPublisher) PublishAsync(ctx context.Context, event *Event) {
}

// Close does nothing (no-op)
func (p *NoOpPublisher) Close() error {
	return nil
}
