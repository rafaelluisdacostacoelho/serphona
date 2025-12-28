package invoke

import (
	"context"
	"time"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// CancelableExecutor wraps an Executor and emits a cancellation event when the context is canceled
// before the inner executor completes streaming.
type CancelableExecutor struct {
	inner Executor
}

// NewCancelableExecutor wraps an executor with cancellation awareness.
func NewCancelableExecutor(inner Executor) *CancelableExecutor {
	return &CancelableExecutor{inner: inner}
}

// Invoke forwards events from the inner executor and injects a cancellation event when ctx is done.
func (c *CancelableExecutor) Invoke(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	in, err := c.inner.Invoke(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make(chan protocol.InvocationEvent)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				out <- protocol.InvocationEvent{
					Type:     protocol.EventError,
					Error:    &protocol.InvocationError{Code: mcperrors.ErrCancelled, Message: "invocation cancelled", Details: ctx.Err()},
					Progress: &protocol.Progress{Stage: "cancelled", Timestamp: time.Now().UTC()},
				}
				return
			case evt, ok := <-in:
				if !ok {
					return
				}
				out <- evt
			}
		}
	}()
	return out, nil
}
