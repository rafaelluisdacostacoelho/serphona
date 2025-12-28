package invoke

import (
	"context"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

func TestOutputLimitBlocksLargeResult(t *testing.T) {
	inner := NewStaticExecutor(map[string]Handler{
		"echo": func(ctx context.Context, _ protocol.InvocationRequest) (protocol.InvocationEvent, error) {
			return protocol.InvocationEvent{Type: protocol.EventResult, Data: []byte("123456")}, nil
		},
	})
	ol := NewOutputLimitExecutor(inner, 4)

	ch, err := ol.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if evt := <-ch; evt.Type != protocol.EventError {
		t.Fatalf("expected error, got %+v", evt)
	}
}

func TestOutputLimitPassesSmallResult(t *testing.T) {
	inner := NewStaticExecutor(map[string]Handler{
		"echo": func(ctx context.Context, _ protocol.InvocationRequest) (protocol.InvocationEvent, error) {
			return protocol.InvocationEvent{Type: protocol.EventResult, Data: []byte("123")}, nil
		},
	})
	ol := NewOutputLimitExecutor(inner, 4)

	ch, err := ol.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if evt := <-ch; evt.Type != protocol.EventResult {
		t.Fatalf("expected result, got %+v", evt)
	}
}
