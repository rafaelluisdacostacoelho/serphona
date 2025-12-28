package invoke

import (
	"context"
	"encoding/json"
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
