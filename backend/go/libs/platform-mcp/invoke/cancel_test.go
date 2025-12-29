package invoke

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type passThroughExecutor struct {
	ch <-chan protocol.InvocationEvent
}

func (p passThroughExecutor) Invoke(_ context.Context, _ protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	return p.ch, nil
}

func TestCancelableExecutorForwardsEvents(t *testing.T) {
	innerCh := make(chan protocol.InvocationEvent, 1)
	innerCh <- protocol.InvocationEvent{Type: protocol.EventResult}
	close(innerCh)

	ce := NewCancelableExecutor(passThroughExecutor{ch: innerCh})
	out, err := ce.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if evt := <-out; evt.Type != protocol.EventResult {
		t.Fatalf("unexpected event: %+v", evt)
	}
}

func TestCancelableExecutorEmitsCancelOnContextDone(t *testing.T) {
	innerCh := make(chan protocol.InvocationEvent)
	ce := NewCancelableExecutor(passThroughExecutor{ch: innerCh})

	ctx, cancel := context.WithCancel(context.Background())
	out, err := ce.Invoke(ctx, protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}

	cancel()

	select {
	case evt := <-out:
		if evt.Type != protocol.EventError || evt.Error == nil || evt.Error.Code != "cancelled" {
			t.Fatalf("expected cancel event, got %+v", evt)
		}
		if evt.Progress == nil || evt.Progress.Stage != "cancelled" || evt.Progress.Timestamp.IsZero() {
			t.Fatalf("expected progress metadata on cancel, got %+v", evt.Progress)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for cancel event")
	}
}

type errorExecutor struct{}

func (errorExecutor) Invoke(_ context.Context, _ protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	return nil, errors.New("boom")
}

func TestCancelableExecutorPropagatesInnerError(t *testing.T) {
	ce := NewCancelableExecutor(errorExecutor{})

	if _, err := ce.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)}); err == nil {
		t.Fatalf("expected inner error to propagate")
	}
}
