//go:build ignore
// +build ignore

// Deprecated duplicate of stream.go; build-ignored to avoid symbol clashes.
package invoke

import (
	"context"
	"fmt"
	"time"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// StreamingHandler streams progress/result events into the provided channel.
// Implementations must return once finished or when ctx is canceled.
type StreamingHandler func(ctx context.Context, req protocol.InvocationRequest, out chan<- protocol.InvocationEvent) error

// StreamingExecutor executes a StreamingHandler, propagating cancel and panics as events.
type StreamingExecutor struct {
	handler StreamingHandler
}

// NewStreamingExecutor builds an executor for streaming handlers.
func NewStreamingExecutor(handler StreamingHandler) *StreamingExecutor {
	return &StreamingExecutor{handler: handler}
}

// Invoke runs the handler, forwarding its events and emitting cancellation/error frames as needed.
func (s *StreamingExecutor) Invoke(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	if err := protocol.ValidateInvocation(req); err != nil {
		return nil, err
	}
	out := make(chan protocol.InvocationEvent)

	go func() {
		defer close(out)

		errCh := make(chan error, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					errCh <- fmt.Errorf("panic: %v", r)
				}
			}()
			errCh <- s.handler(ctx, req, out)
		}()

		select {
		case <-ctx.Done():
			out <- protocol.InvocationEvent{
				Type:     protocol.EventError,
				Error:    &protocol.InvocationError{Code: mcperrors.ErrCancelled, Message: "invocation cancelled", Details: ctx.Err()},
				Progress: &protocol.Progress{Stage: "cancelled", Timestamp: time.Now().UTC()},
			}
			return
		case err := <-errCh:
			if err != nil {
				out <- protocol.InvocationEvent{Type: protocol.EventError, Error: &protocol.InvocationError{Code: mcperrors.ErrInternal, Message: err.Error()}}
			}
			return
		}
	}()

	return out, nil
}
