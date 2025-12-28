package invoke

import (
	"context"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// MultiObserver fans out events to multiple observers.
type MultiObserver struct {
	observers []Observer
}

// NewMultiObserver builds a fan-out observer.
func NewMultiObserver(obs ...Observer) *MultiObserver {
	return &MultiObserver{observers: obs}
}

// OnInvocationEvent forwards to all observers.
func (m *MultiObserver) OnInvocationEvent(ctx context.Context, req protocol.InvocationRequest, evt protocol.InvocationEvent, invokeErr error, elapsed time.Duration) {
	for _, o := range m.observers {
		if o != nil {
			o.OnInvocationEvent(ctx, req, evt, invokeErr, elapsed)
		}
	}
}
