package invoke

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// StreamingHandler can emit progress and result events on a channel.
type StreamingHandler func(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error)

// StreamingExecutor dispatches to streaming handlers keyed by tool name (lowercase).
type StreamingExecutor struct {
	handlers map[string]StreamingHandler
}

// NewStreamingExecutor builds an executor from streaming handlers.
func NewStreamingExecutor(handlers map[string]StreamingHandler) *StreamingExecutor {
	return &StreamingExecutor{handlers: handlers}
}

// Invoke dispatches to a streaming handler; handlers are responsible for closing the channel.
func (s *StreamingExecutor) Invoke(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	if err := protocol.ValidateInvocation(req); err != nil {
		return nil, err
	}
	h, ok := s.handlers[strings.ToLower(req.Tool.Name)]
	if !ok {
		return nil, errors.New("tool handler not found")
	}
	ch, err := safeCallHandler(ctx, h, req)
	if err != nil {
		return errorStream(err), nil
	}
	if ch == nil {
		return errorStream(errors.New("handler returned nil channel")), nil
	}
	return wrapWithCancel(ctx, ch), nil
}

func safeCallHandler(ctx context.Context, h StreamingHandler, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	var (
		ch  <-chan protocol.InvocationEvent
		err error
	)
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	ch, err = h(ctx, req)
	return ch, err
}

func wrapWithCancel(ctx context.Context, in <-chan protocol.InvocationEvent) <-chan protocol.InvocationEvent {
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
	return out
}

func errorStream(err error) <-chan protocol.InvocationEvent {
	out := make(chan protocol.InvocationEvent, 1)
	out <- protocol.InvocationEvent{Type: protocol.EventError, Error: &protocol.InvocationError{Code: mcperrors.ErrInternal, Message: err.Error()}}
	close(out)
	return out
}
