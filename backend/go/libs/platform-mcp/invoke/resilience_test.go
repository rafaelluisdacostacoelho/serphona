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

func TestResilientExecutorHonorsCircuitOpen(t *testing.T) {
	innerCalls := 0
	inner := ExecutorFunc(func(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
		innerCalls++
		return nil, errors.New("boom")
	})
	cb := NewCircuitBreaker(1, time.Minute)
	cb.openUntil = time.Now().Add(time.Hour) // force open
	r := NewResilientExecutor(inner, ResilientConfig{MaxRetries: 1, Breaker: cb})

	if _, err := r.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)}); err == nil {
		t.Fatalf("expected circuit open error")
	}
	if innerCalls != 0 {
		t.Fatalf("inner should not be called when circuit open")
	}
}

func TestResilientExecutorBackoffFunctionInvoked(t *testing.T) {
	inner := &failingExecutor{maxFail: 2}
	backoffCalls := 0
	r := NewResilientExecutor(inner, ResilientConfig{
		MaxRetries: 2,
		Backoff: func(attempt int) time.Duration {
			backoffCalls++
			return time.Millisecond
		},
		Sleep: func(time.Duration) {},
	})

	_, _ = r.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if backoffCalls == 0 {
		t.Fatalf("expected backoff function to be called")
	}
}

func TestResilientExecutorNoRetryOnSuccess(t *testing.T) {
	attempts := 0
	inner := ExecutorFunc(func(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
		attempts++
		ch := make(chan protocol.InvocationEvent, 1)
		ch <- protocol.InvocationEvent{Type: protocol.EventResult}
		return ch, nil
	})
	slept := 0
	r := NewResilientExecutor(inner, ResilientConfig{MaxRetries: 3, Sleep: func(time.Duration) { slept++ }})

	ch, err := r.Invoke(context.Background(), protocol.InvocationRequest{Version: protocol.CurrentVersion, TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}, Input: []byte(`{}`)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected single attempt, got %d", attempts)
	}
	if slept != 0 {
		t.Fatalf("expected no sleep when no retry, got %d", slept)
	}
	if evt := <-ch; evt.Type != protocol.EventResult {
		t.Fatalf("unexpected event: %+v", evt)
	}
}
