package invoke

import (
	"context"
	"errors"
	"testing"
	"time"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

func TestStreamingExecutorForwardsProgressAndResult(t *testing.T) {
	handler := func(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
		ch := make(chan protocol.InvocationEvent, 3)
		ch <- protocol.InvocationEvent{Type: protocol.EventProgress, Progress: &protocol.Progress{Stage: "p1"}}
		ch <- protocol.InvocationEvent{Type: protocol.EventProgress, Progress: &protocol.Progress{Stage: "p2"}}
		ch <- protocol.InvocationEvent{Type: protocol.EventResult, Data: []byte("ok")}
		close(ch)
		return ch, nil
	}

	exec := NewStreamingExecutor(map[string]StreamingHandler{"echo": handler})
	req := protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte("hi")}

	ch, err := exec.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("invoke error: %v", err)
	}

	events := collectEvents(ch)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Progress.Stage != "p1" || events[1].Progress.Stage != "p2" || events[2].Type != protocol.EventResult {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestStreamingExecutorCancelsDownstream(t *testing.T) {
	handler := func(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
		ch := make(chan protocol.InvocationEvent)
		go func() {
			defer close(ch)
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
				ch <- protocol.InvocationEvent{Type: protocol.EventResult, Data: []byte("late")}
			}
		}()
		return ch, nil
	}

	exec := NewStreamingExecutor(map[string]StreamingHandler{"echo": handler})
	ctx, cancel := context.WithCancel(context.Background())
	req := protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte("hi")}

	ch, err := exec.Invoke(ctx, req)
	if err != nil {
		t.Fatalf("invoke error: %v", err)
	}
	cancel()

	events := collectEvents(ch)
	if len(events) != 1 {
		t.Fatalf("expected cancel event, got %d", len(events))
	}
	evt := events[0]
	if evt.Error == nil || evt.Error.Code != mcperrors.ErrCancelled {
		t.Fatalf("expected cancelled error, got %#v", evt)
	}
	if evt.Progress == nil || evt.Progress.Stage != "cancelled" {
		t.Fatalf("expected cancelled progress, got %#v", evt.Progress)
	}
}

func TestStreamingExecutorPropagatesHandlerError(t *testing.T) {
	handler := func(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
		return nil, errors.New("boom")
	}
	exec := NewStreamingExecutor(map[string]StreamingHandler{"echo": handler})
	req := protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte("hi")}

	ch, err := exec.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("invoke error: %v", err)
	}
	events := collectEvents(ch)
	if len(events) != 1 || events[0].Error == nil || events[0].Error.Code != mcperrors.ErrInternal {
		t.Fatalf("expected internal error event, got %#v", events)
	}
}

func TestStreamingExecutorHandlesPanics(t *testing.T) {
	handler := func(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
		panic("oh no")
	}
	exec := NewStreamingExecutor(map[string]StreamingHandler{"echo": handler})
	req := protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte("hi")}

	ch, err := exec.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("invoke error: %v", err)
	}
	events := collectEvents(ch)
	if len(events) != 1 || events[0].Error == nil || events[0].Error.Code != mcperrors.ErrInternal {
		t.Fatalf("expected internal error from panic, got %#v", events)
	}
}

func collectEvents(ch <-chan protocol.InvocationEvent) []protocol.InvocationEvent {
	var evts []protocol.InvocationEvent
	for evt := range ch {
		evts = append(evts, evt)
	}
	return evts
}
