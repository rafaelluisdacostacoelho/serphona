package invoke

import (
	"context"
	"errors"
	"strings"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// Executor executes tool invocations and streams events.
type Executor interface {
	Invoke(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error)
}

// NoopExecutor returns an immediate error event for unsupported tools.
type NoopExecutor struct{}

// Invoke implements Executor for NoopExecutor.
func (NoopExecutor) Invoke(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	if err := protocol.ValidateInvocation(req); err != nil {
		return nil, err
	}
	ch := make(chan protocol.InvocationEvent, 1)
	ch <- protocol.InvocationEvent{
		Type:  protocol.EventError,
		Error: &protocol.InvocationError{Code: "not_implemented", Message: "executor not configured"},
	}
	close(ch)
	return ch, nil
}

// StaticExecutor routes to a map of tool handlers.
type StaticExecutor struct {
	handlers map[string]Handler
}

// Handler runs a tool given input bytes and returns a result payload.
type Handler func(ctx context.Context, req protocol.InvocationRequest) (protocol.InvocationEvent, error)

// NewStaticExecutor builds an executor from handlers keyed by tool name (lowercase).
func NewStaticExecutor(handlers map[string]Handler) *StaticExecutor {
	return &StaticExecutor{handlers: handlers}
}

// Invoke dispatches to a registered handler and streams a single result or error.
func (s *StaticExecutor) Invoke(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	if err := protocol.ValidateInvocation(req); err != nil {
		return nil, err
	}
	ch := make(chan protocol.InvocationEvent, 1)
	h, ok := s.handlers[strings.ToLower(req.Tool.Name)]
	if !ok {
		return nil, errors.New("tool handler not found")
	}
	go func() {
		defer close(ch)
		res, err := h(ctx, req)
		if err != nil {
			ch <- protocol.InvocationEvent{Type: protocol.EventError, Error: &protocol.InvocationError{Code: "handler_error", Message: err.Error()}}
			return
		}
		ch <- res
	}()
	return ch, nil
}
