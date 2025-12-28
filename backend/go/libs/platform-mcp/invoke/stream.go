package invoke

import (
	"context"
	"errors"
	"strings"

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
	return h(ctx, req)
}
