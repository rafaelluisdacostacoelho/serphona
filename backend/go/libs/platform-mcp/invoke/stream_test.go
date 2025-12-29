package invoke

import (
	"context"
	"testing"
	"time"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

func TestStreamingExecutorEmitsProgressAndResult(t *testing.T) {
	h := func(ctx context.Context, _ protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
		ch := make(chan protocol.InvocationEvent, 2)
		ch <- protocol.InvocationEvent{Type: protocol.EventProgress, Progress: &protocol.Progress{Stage: "start"}}
		ch <- protocol.InvocationEvent{Type: protocol.EventResult}
		close(ch)
		return ch, nil
	}
	exec := NewStreamingExecutor(map[string]StreamingHandler{"echo": h})
	ch, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if evt := <-ch; evt.Type != protocol.EventProgress {
		t.Fatalf("expected progress, got %+v", evt)
	}
	if evt := <-ch; evt.Type != protocol.EventResult {
		t.Fatalf("expected result, got %+v", evt)
	}
}

func TestStreamingExecutorHandlerError(t *testing.T) {
	h := func(ctx context.Context, _ protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
		return nil, context.Canceled
	}
	exec := NewStreamingExecutor(map[string]StreamingHandler{"echo": h})
	ch, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke error: %v", err)
	}
	evts := collectEvents(ch)
	if len(evts) != 1 || evts[0].Error == nil {
		t.Fatalf("expected error event, got %#v", evts)
	}
}

func TestStreamingExecutorValidationError(t *testing.T) {
	exec := NewStreamingExecutor(map[string]StreamingHandler{})
	if _, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: "", TenantID: "", Tool: protocol.ToolRef{Name: ""}, Input: []byte{}}); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestStreamingExecutorValidates(t *testing.T) {
	exec := NewStreamingExecutor(map[string]StreamingHandler{})
	if _, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "", Tool: protocol.ToolRef{Name: ""}, Input: []byte{}}); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestStreamingExecutorCanBeCancelledDownstream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := func(ctx context.Context, _ protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
		ch := make(chan protocol.InvocationEvent)
		go func() {
			defer close(ch)
			<-time.After(time.Second)
		}()
		return ch, nil
	}
	exec := NewStreamingExecutor(map[string]StreamingHandler{"echo": h})
	ch, err := exec.Invoke(ctx, protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	cancel()
	select {
	case evt, ok := <-ch:
		if !ok {
			t.Fatalf("expected cancel event before close")
		}
		if evt.Error == nil || evt.Error.Code != mcperrors.ErrCancelled {
			t.Fatalf("expected cancelled event, got %#v", evt)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for cancel event")
	}
	select {
	case <-ch:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for channel close")
	}
}

func TestStreamingExecutorRejectsNilChannel(t *testing.T) {
	exec := NewStreamingExecutor(map[string]StreamingHandler{
		"echo": func(ctx context.Context, _ protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
			return nil, nil
		},
	})

	ch, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke err: %v", err)
	}
	events := collectEvents(ch)
	if len(events) != 1 || events[0].Error == nil || events[0].Error.Code != mcperrors.ErrInternal {
		t.Fatalf("expected internal error event for nil channel, got %+v", events)
	}
}

func TestStreamingExecutorMissingHandlerErrors(t *testing.T) {
	exec := NewStreamingExecutor(map[string]StreamingHandler{})
	_, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "missing"}, Input: []byte(`{}`)})
	if err == nil {
		t.Fatalf("expected error for missing handler")
	}
}
