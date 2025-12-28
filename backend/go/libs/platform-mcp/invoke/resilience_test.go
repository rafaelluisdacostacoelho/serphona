package invoke

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type failingExecutor struct {
	failures int32
	maxFail  int32
}

func (f *failingExecutor) Invoke(_ context.Context, _ protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	if atomic.AddInt32(&f.failures, 1) <= f.maxFail {
		return nil, errors.New("boom")
	}
	ch := make(chan protocol.InvocationEvent, 1)
	ch <- protocol.InvocationEvent{Type: protocol.EventResult}
	return ch, nil
}

func TestResilientExecutorRetriesThenSucceeds(t *testing.T) {
	inner := &failingExecutor{maxFail: 1}
	slept := 0
	r := NewResilientExecutor(inner, ResilientConfig{MaxRetries: 2, Sleep: func(d time.Duration) { slept++ }})

	ch, err := r.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if slept == 0 {
		t.Fatalf("expected backoff sleep on retry")
	}
	if evt := <-ch; evt.Type != protocol.EventResult {
		t.Fatalf("unexpected event: %+v", evt)
	}
}

func TestResilientExecutorStopsAtMaxRetries(t *testing.T) {
	inner := &failingExecutor{maxFail: 5}
	attempts := 0
	r := NewResilientExecutor(inner, ResilientConfig{MaxRetries: 1, Sleep: func(d time.Duration) { attempts++ }})

	ch, err := r.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err == nil || ch != nil {
		t.Fatalf("expected failure after retries")
	}
	if attempts == 0 {
		t.Fatalf("expected at least one sleep")
	}
}

func TestCircuitBreakerOpensAfterFailuresAndResets(t *testing.T) {
	breaker := NewCircuitBreaker(2, time.Minute)
	breaker.now = func() time.Time { return time.Unix(0, 0) }

	if !breaker.Allow() {
		t.Fatalf("expected allow initially")
	}
	breaker.Failure()
	if !breaker.Allow() {
		t.Fatalf("threshold not reached yet")
	}
	breaker.Failure()
	if breaker.Allow() {
		t.Fatalf("circuit should be open")
	}

	breaker.now = func() time.Time { return time.Unix(int64(time.Minute.Seconds()+1), 0) }
	if !breaker.Allow() {
		t.Fatalf("expected breaker to reset after window")
	}
	breaker.Success()
	if !breaker.Allow() {
		t.Fatalf("breaker should stay closed after success")
	}
}
