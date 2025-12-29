package invoke

import (
	"context"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type captureDeadlineExec struct {
	deadline time.Time
}

func (c *captureDeadlineExec) Invoke(ctx context.Context, _ protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	if dl, ok := ctx.Deadline(); ok {
		c.deadline = dl
	}
	ch := make(chan protocol.InvocationEvent, 1)
	ch <- protocol.InvocationEvent{Type: protocol.EventResult}
	return ch, nil
}

func TestGuardExecutorRejectsLargePayload(t *testing.T) {
	guard := NewGuardExecutor(NoopExecutor{}, GuardConfig{MaxBodyBytes: 4})
	_, err := guard.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte("12345")})
	if err == nil {
		t.Fatalf("expected size rejection")
	}
}

func TestGuardExecutorAppliesTimeoutCap(t *testing.T) {
	capture := &captureDeadlineExec{}
	guard := NewGuardExecutor(capture, GuardConfig{DefaultTimeout: 100 * time.Millisecond, MaxTimeout: 150 * time.Millisecond})

	start := time.Now()
	_, err := guard.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte("{}"), Timeout: 500 * time.Millisecond})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capture.deadline.IsZero() {
		t.Fatalf("expected deadline to be set")
	}
	if capture.deadline.Sub(start) > 200*time.Millisecond {
		t.Fatalf("deadline not capped, got %v", capture.deadline.Sub(start))
	}
}

func TestGuardExecutorAppliesDefaultTimeout(t *testing.T) {
	capture := &captureDeadlineExec{}
	guard := NewGuardExecutor(capture, GuardConfig{DefaultTimeout: 50 * time.Millisecond})

	start := time.Now()
	_, err := guard.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte("{}")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capture.deadline.IsZero() || capture.deadline.Sub(start) > 100*time.Millisecond {
		t.Fatalf("expected default timeout applied, got deadline %v", capture.deadline)
	}
}
