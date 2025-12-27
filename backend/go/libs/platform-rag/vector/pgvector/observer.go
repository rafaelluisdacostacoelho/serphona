package pgvector

import (
	"context"
	"time"
)

// Observer captures tracing/metrics around store operations.
type Observer interface {
	Trace(ctx context.Context, operation string) (context.Context, func(error))
	RecordLatency(ctx context.Context, operation string, d time.Duration, err error)
}

// NoopObserver is the default when no observer is provided.
type NoopObserver struct{}

func (NoopObserver) Trace(ctx context.Context, _ string) (context.Context, func(error)) {
	return ctx, func(error) {}
}

func (NoopObserver) RecordLatency(context.Context, string, time.Duration, error) { return }

func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
