package invoke

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestBuildExecutorReturnsBaseWhenNoOptions(t *testing.T) {
	base := NewStaticExecutor(map[string]Handler{})
	out := BuildExecutor(base, StackConfig{})
	if out != base {
		t.Fatalf("expected base executor returned when no options")
	}
}

type stackCountingObserver struct{ count int }

func (c *stackCountingObserver) OnInvocationEvent(ctx context.Context, req protocol.InvocationRequest, evt protocol.InvocationEvent, invokeErr error, elapsed time.Duration) {
	c.count++
}

type allowLimiter struct{ calls int }

func (a *allowLimiter) Allow(context.Context, string, string) bool {
	a.calls++
	return true
}

func TestBuildExecutorWithRateLimitResilientAndExtraObservers(t *testing.T) {
	innerCalls := 0
	base := ExecutorFunc(func(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
		innerCalls++
		if innerCalls == 1 {
			return nil, errors.New("fail once")
		}
		ch := make(chan protocol.InvocationEvent, 1)
		ch <- protocol.InvocationEvent{Type: protocol.EventResult}
		return ch, nil
	})

	rate := &allowLimiter{}
	obsPrimary := &stackCountingObserver{}
	obsExtra := &stackCountingObserver{}
	exec := BuildExecutor(base, StackConfig{
		RateLimiter: rate,
		Resilient:   &ResilientConfig{MaxRetries: 1, Sleep: func(time.Duration) {}},
		Observer:    obsPrimary,
		ExtraObservers: []Observer{
			obsExtra,
		},
	})

	ch, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("invoke err: %v", err)
	}
	if evt := <-ch; evt.Type != protocol.EventResult {
		t.Fatalf("expected result event, got %+v", evt)
	}
	if rate.calls == 0 {
		t.Fatalf("rate limiter not called")
	}
	if obsPrimary.count == 0 || obsExtra.count == 0 {
		t.Fatalf("observers not invoked: primary=%d extra=%d", obsPrimary.count, obsExtra.count)
	}
	if innerCalls != 2 {
		t.Fatalf("expected retry via resilient wrapper, got %d calls", innerCalls)
	}
}
