package invoke

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

func TestStaticExecutor(t *testing.T) {
	exec := NewStaticExecutor(map[string]Handler{
		"echo": func(ctx context.Context, req protocol.InvocationRequest) (protocol.InvocationEvent, error) {
			return protocol.InvocationEvent{Type: protocol.EventResult, Data: req.Input}, nil
		},
	})

	ch, err := exec.Invoke(context.Background(), protocol.InvocationRequest{
		Version:  protocol.CurrentVersion,
		TenantID: "t1",
		Tool:     protocol.ToolRef{Name: "echo"},
		Input:    json.RawMessage(`{"msg":"ok"}`),
	})
	if err != nil {
		t.Fatalf("invoke error: %v", err)
	}
	evt := <-ch
	if evt.Type != protocol.EventResult {
		t.Fatalf("expected result event, got %s", evt.Type)
	}
}

func TestStaticExecutorMissingHandler(t *testing.T) {
	exec := NewStaticExecutor(map[string]Handler{})
	if _, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "missing"}, Input: []byte(`{}`)}); err == nil {
		t.Fatalf("expected error for missing handler")
	}
}

func TestStaticExecutorHandlerError(t *testing.T) {
	exec := NewStaticExecutor(map[string]Handler{
		"echo": func(ctx context.Context, _ protocol.InvocationRequest) (protocol.InvocationEvent, error) {
			return protocol.InvocationEvent{}, errors.New("boom")
		},
	})

	ch, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke err: %v", err)
	}
	evt := <-ch
	if evt.Type != protocol.EventError || evt.Error == nil {
		t.Fatalf("expected error event, got %+v", evt)
	}
}

func TestStaticExecutorValidationError(t *testing.T) {
	exec := NewStaticExecutor(map[string]Handler{"echo": func(ctx context.Context, req protocol.InvocationRequest) (protocol.InvocationEvent, error) {
		return protocol.InvocationEvent{Type: protocol.EventResult}, nil
	}})

	if _, err := exec.Invoke(context.Background(), protocol.InvocationRequest{TenantID: "", Version: "", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)}); err == nil {
		t.Fatalf("expected validation error for invalid invocation")
	}
}

func TestNoopExecutorValidationError(t *testing.T) {
	noop := NoopExecutor{}
	if _, err := noop.Invoke(context.Background(), protocol.InvocationRequest{Version: "", TenantID: "", Tool: protocol.ToolRef{Name: ""}}); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestNoopExecutorReturnsNotImplementedEvent(t *testing.T) {
	noop := NoopExecutor{}
	ch, err := noop.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "missing"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	evt := <-ch
	if evt.Type != protocol.EventError || evt.Error == nil || evt.Error.Code != "not_implemented" {
		t.Fatalf("expected not_implemented error event, got %+v", evt)
	}
}
