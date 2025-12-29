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

func TestRateLimitExecutorAllowsWhenLimiterNil(t *testing.T) {
	inner := &stubExecutor{}
	exec := NewRateLimitExecutor(inner, nil, 0)

	if _, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}); err != nil {
		t.Fatalf("expected allow with nil limiter: %v", err)
	}
	if !inner.called {
		t.Fatalf("inner should be called when limiter nil")
	}
}

func TestRateLimitExecutorAllowsWhenLimiterApproves(t *testing.T) {
	limiter := &stubLimiter{allowed: true}
	inner := &stubExecutor{}
	exec := NewRateLimitExecutor(inner, limiter, 0)

	if _, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}); err != nil {
		t.Fatalf("expected allow when limiter approves: %v", err)
	}
	if !inner.called {
		t.Fatalf("inner should be invoked when limiter allows")
	}
}

func TestRateLimitExecutorCooldownSleeps(t *testing.T) {
	limiter := &stubLimiter{allowed: false}
	inner := &stubExecutor{}
	exec := NewRateLimitExecutor(inner, limiter, 1*time.Millisecond)

	if _, err := exec.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}); err == nil {
		t.Fatalf("expected rate limit error")
	}
	if inner.called {
		t.Fatalf("inner should not be called on deny")
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

func TestRateLimitExecutorCooldownHonorsContext(t *testing.T) {
	limiter := &stubLimiter{allowed: false}
	inner := &stubExecutor{}
	exec := NewRateLimitExecutor(inner, limiter, 5*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := exec.Invoke(ctx, protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}); err == nil {
		t.Fatalf("expected rate limit error even when context canceled")
	}
	if inner.called {
		t.Fatalf("inner executor should not be called on deny")
	}
}

func TestMemoryRateLimiterDefaultsAndRefill(t *testing.T) {
	lim := NewMemoryRateLimiter(0, 0)
	if lim.rate != 1 || lim.burst != 1 {
		t.Fatalf("expected defaults set, got rate=%v burst=%v", lim.rate, lim.burst)
	}

	lim.clock = func() time.Time { return time.Unix(0, 0) }
	if !lim.Allow(context.Background(), "t1", "tool") {
		t.Fatalf("expected first allow")
	}
	if lim.Allow(context.Background(), "t1", "tool") {
		t.Fatalf("expected second call to be denied with empty bucket")
	}
	lim.clock = func() time.Time { return time.Unix(1, 0) }
	if !lim.Allow(context.Background(), "t1", "tool") {
		t.Fatalf("expected allow after refill")
	}
}

func TestMinHelper(t *testing.T) {
	if got := min(1, 2); got != 1 {
		t.Fatalf("expected min 1, got %v", got)
	}
	if got := min(3, 2); got != 2 {
		t.Fatalf("expected min 2, got %v", got)
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
