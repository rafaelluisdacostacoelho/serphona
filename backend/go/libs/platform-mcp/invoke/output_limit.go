package invoke

import (
	"context"
	"errors"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// OutputLimitExecutor enforces maximum output size for result events.
type OutputLimitExecutor struct {
	inner   Executor
	maxSize int64
}

// NewOutputLimitExecutor wraps an executor enforcing result data max bytes (0 disables).
func NewOutputLimitExecutor(inner Executor, maxSize int64) *OutputLimitExecutor {
	return &OutputLimitExecutor{inner: inner, maxSize: maxSize}
}

// Invoke forwards events and converts oversized results into errors.
func (o *OutputLimitExecutor) Invoke(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	ch, err := o.inner.Invoke(ctx, req)
	if err != nil || o.maxSize <= 0 {
		return ch, err
	}
	out := make(chan protocol.InvocationEvent)
	go func() {
		defer close(out)
		for evt := range ch {
			if evt.Type == protocol.EventResult && len(evt.Data) > int(o.maxSize) {
				out <- protocol.InvocationEvent{Type: protocol.EventError, Error: &protocol.InvocationError{Code: mcperrors.ErrInvalidRequest, Message: "result exceeds max output bytes", Details: errors.New("max output exceeded")}}
				return
			}
			out <- evt
		}
	}()
	return out, nil
}
