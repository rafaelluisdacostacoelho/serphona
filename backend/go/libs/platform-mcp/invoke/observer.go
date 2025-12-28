package invoke

import (
	"context"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// Observer receives invocation events for metrics/audit purposes.
type Observer interface {
	OnInvocationEvent(ctx context.Context, req protocol.InvocationRequest, evt protocol.InvocationEvent, invokeErr error, elapsed time.Duration)
}

// ObservedExecutor wraps an Executor, timing each invocation and forwarding events to the observer.
type ObservedExecutor struct {
	inner    Executor
	observer Observer
}

// NewObservedExecutor creates an observed executor.
func NewObservedExecutor(inner Executor, observer Observer) *ObservedExecutor {
	return &ObservedExecutor{inner: inner, observer: observer}
}

// Invoke forwards to the inner executor while emitting observer callbacks for errors and stream events.
func (o *ObservedExecutor) Invoke(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	start := time.Now()
	ch, err := o.inner.Invoke(ctx, req)
	if err != nil {
		if o.observer != nil {
			o.observer.OnInvocationEvent(ctx, req, protocol.InvocationEvent{}, err, time.Since(start))
		}
		return nil, err
	}
	if o.observer == nil {
		return ch, nil
	}
	out := make(chan protocol.InvocationEvent)
	go func() {
		defer close(out)
		for evt := range ch {
			elapsed := time.Since(start)
			o.observer.OnInvocationEvent(ctx, req, evt, nil, elapsed)
			out <- evt
		}
	}()
	return out, nil
}
