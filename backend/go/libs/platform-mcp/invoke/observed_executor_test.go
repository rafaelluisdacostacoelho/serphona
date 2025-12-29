package invoke

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type obsStubExecutor struct {
	ch  chan protocol.InvocationEvent
	err error
}

func (s *obsStubExecutor) Invoke(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.ch, nil
}

type countingObserver struct {
	count   int
	lastErr error
}

func (c *countingObserver) OnInvocationEvent(ctx context.Context, req protocol.InvocationRequest, evt protocol.InvocationEvent, invokeErr error, elapsed time.Duration) {
	c.count++
	c.lastErr = invokeErr
}

func TestObservedExecutorPropagatesErrorOnce(t *testing.T) {
	obs := &countingObserver{}
	exec := &obsStubExecutor{err: errors.New("boom")}
	oe := NewObservedExecutor(exec, obs)

	if _, err := oe.Invoke(context.Background(), protocol.InvocationRequest{}); err == nil {
		t.Fatalf("expected error")
	}
	if obs.count != 1 || obs.lastErr == nil {
		t.Fatalf("observer should receive error")
	}
}

func TestObservedExecutorForwardsBufferedEvents(t *testing.T) {
	obs := &countingObserver{}
	ch := make(chan protocol.InvocationEvent, 2)
	ch <- protocol.InvocationEvent{Type: protocol.EventProgress}
	ch <- protocol.InvocationEvent{Type: protocol.EventResult}
	close(ch)

	exec := &obsStubExecutor{ch: ch}
	oe := NewObservedExecutor(exec, obs)

	out, err := oe.Invoke(context.Background(), protocol.InvocationRequest{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// drain out
	for range out {
	}
	if obs.count != 2 {
		t.Fatalf("observer should see all events, got %d", obs.count)
	}
}
