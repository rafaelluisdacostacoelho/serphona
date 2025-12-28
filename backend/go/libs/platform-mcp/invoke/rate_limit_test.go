package invoke

import (
	"context"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

func TestMemoryRateLimiterAllowsWithinBurst(t *testing.T) {
	lim := NewMemoryRateLimiter(1, 2)
	lim.clock = func() time.Time { return time.Unix(0, 0) }

	if !lim.Allow(context.Background(), "t1", "echo") {
		t.Fatalf("expected first allow")
	}
	if !lim.Allow(context.Background(), "t1", "echo") {
		t.Fatalf("expected second allow within burst")
	}
	if lim.Allow(context.Background(), "t1", "echo") {
		t.Fatalf("expected third to be denied")
	}
	// advance time to refill 1 token
	lim.clock = func() time.Time { return time.Unix(1, 0) }
	if !lim.Allow(context.Background(), "t1", "echo") {
		t.Fatalf("expected allow after refill")
	}
}

func TestRateLimitExecutorBlocksAndReturnsError(t *testing.T) {
	lim := NewMemoryRateLimiter(0, 1)
	lim.clock = func() time.Time { return time.Unix(0, 0) }
	base := NewStaticExecutor(map[string]Handler{"echo": func(ctx context.Context, _ protocol.InvocationRequest) (protocol.InvocationEvent, error) {
		return protocol.InvocationEvent{Type: protocol.EventResult}, nil
	}})

	exec := NewRateLimitExecutor(base, lim, 0)
	// consume initial token
	if _, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)}); err != nil {
		t.Fatalf("unexpected first err: %v", err)
	}
	if _, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)}); err == nil {
		t.Fatalf("expected rate limit error")
	}
}

func TestRateLimitExecutorDoesNotCallInnerOnDeny(t *testing.T) {
	limiter := &stubLimiter{allowed: false}
	inner := &stubExecutor{}
	exec := NewRateLimitExecutor(inner, limiter, 0)

	_, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}})
	if err == nil {
		t.Fatalf("expected rate limited error")
	}
	if inner.called {
		t.Fatalf("inner executor should not be called when limited")
	}
}

func TestMemoryRateLimiterIsolatedPerTool(t *testing.T) {
	lim := NewMemoryRateLimiter(1, 1)
	lim.clock = func() time.Time { return time.Unix(0, 0) }

	if !lim.Allow(context.Background(), "tenant", "echo") {
		t.Fatalf("expected first echo allow")
	}
	if lim.Allow(context.Background(), "tenant", "echo") {
		t.Fatalf("expected second echo deny")
	}
	if !lim.Allow(context.Background(), "tenant", "calc") {
		t.Fatalf("expected independent bucket per tool")
	}
}

type stubLimiter struct {
	allowed bool
}

func (s *stubLimiter) Allow(context.Context, string, string) bool { return s.allowed }

type stubExecutor struct {
	called bool
}

func (s *stubExecutor) Invoke(_ context.Context, _ protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	s.called = true
	return nil, nil
}
