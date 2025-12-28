package invoke

import (
	"context"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type dummySink struct{ calls int }

func (d *dummySink) IncCounter(_ string, _ map[string]string)                  { d.calls++ }
func (d *dummySink) ObserveHistogram(_ string, _ float64, _ map[string]string) { d.calls++ }

type dummyAuditSink struct{ calls int }

func (d *dummyAuditSink) Write(_ AuditRecord) error { d.calls++; return nil }

func TestBuildExecutorWiresObservers(t *testing.T) {
	metricsSink := &dummySink{}
	auditSink := &dummyAuditSink{}

	base := NewStaticExecutor(map[string]Handler{
		"echo": func(ctx context.Context, _ protocol.InvocationRequest) (protocol.InvocationEvent, error) {
			return protocol.InvocationEvent{Type: protocol.EventResult}, nil
		},
	})

	exec := BuildExecutor(base, StackConfig{
		MetricsSink:    metricsSink,
		AuditSink:      auditSink,
		LabelEnrichers: []LabelEnricher{CacheHitEnricher},
		EnableCancel:   true,
		MaxOutput:      1024,
	})

	ctx := WithCacheHit(context.Background(), "true")
	ch, err := exec.Invoke(ctx, protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if evt := <-ch; evt.Type != protocol.EventResult {
		t.Fatalf("unexpected event: %+v", evt)
	}

	// Metrics sink should have been called twice (counter + hist), audit once.
	if metricsSink.calls == 0 {
		t.Fatalf("expected metrics sink calls")
	}
	if auditSink.calls == 0 {
		t.Fatalf("expected audit sink calls")
	}
}

// Ensure stack respects guard and output limit ordering.
func TestBuildExecutorRespectsGuardAndOutputLimit(t *testing.T) {
	base := NewStaticExecutor(map[string]Handler{
		"echo": func(ctx context.Context, _ protocol.InvocationRequest) (protocol.InvocationEvent, error) {
			return protocol.InvocationEvent{Type: protocol.EventResult, Data: []byte("123456")}, nil
		},
	})
	exec := BuildExecutor(base, StackConfig{
		Guard:     &GuardConfig{MaxBodyBytes: 2},
		MaxOutput: 4,
	})
	if _, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte("123")}); err == nil {
		t.Fatalf("expected guard to reject large input")
	}

	// Now small input but output limit hit.
	ch, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte("1")})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if evt := <-ch; evt.Type != protocol.EventError {
		t.Fatalf("expected output limit error, got %+v", evt)
	}
}
