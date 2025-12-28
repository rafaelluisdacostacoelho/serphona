package invoke

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// ResilientExecutor wraps an Executor with retry/backoff and optional circuit breaker.
// It retries when the wrapped executor returns an error before streaming begins.
type ResilientExecutor struct {
	inner      Executor
	maxRetries int
	backoff    func(attempt int) time.Duration
	breaker    *CircuitBreaker
	sleep      func(time.Duration)
}

// ResilientConfig defines tuning knobs for ResilientExecutor.
type ResilientConfig struct {
	MaxRetries int
	Backoff    func(attempt int) time.Duration
	Breaker    *CircuitBreaker
	Sleep      func(time.Duration)
}

// NewResilientExecutor builds a retrying executor with optional circuit breaker.
func NewResilientExecutor(inner Executor, cfg ResilientConfig) *ResilientExecutor {
	maxRetries := cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	if maxRetries == 0 {
		maxRetries = 2
	}
	backoff := cfg.Backoff
	if backoff == nil {
		backoff = func(attempt int) time.Duration {
			base := 50 * time.Millisecond
			return base << attempt
		}
	}
	slp := cfg.Sleep
	if slp == nil {
		slp = time.Sleep
	}
	return &ResilientExecutor{
		inner:      inner,
		maxRetries: maxRetries,
		backoff:    backoff,
		breaker:    cfg.Breaker,
		sleep:      slp,
	}
}

// Invoke wraps the inner executor with retries/backoff and circuit breaker gating.
func (r *ResilientExecutor) Invoke(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	if r.breaker != nil && !r.breaker.Allow() {
		return nil, errors.New("circuit open")
	}

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		ch, err := r.inner.Invoke(ctx, req)
		if err == nil {
			if r.breaker != nil {
				r.breaker.Success()
			}
			return ch, nil
		}
		if r.breaker != nil {
			r.breaker.Failure()
		}
		if attempt == r.maxRetries {
			return nil, err
		}
		r.sleep(r.backoff(attempt))
	}
	return nil, errors.New("unreachable")
}

// CircuitBreaker tracks failures and opens after threshold until resetAfter elapses.
type CircuitBreaker struct {
	threshold  int
	resetAfter time.Duration
	mu         sync.Mutex
	failures   int
	openUntil  time.Time
	now        func() time.Time
}

// NewCircuitBreaker constructs a simple time-based breaker.
func NewCircuitBreaker(threshold int, resetAfter time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if resetAfter <= 0 {
		resetAfter = 30 * time.Second
	}
	return &CircuitBreaker{threshold: threshold, resetAfter: resetAfter, now: time.Now}
}

// Allow reports whether calls are permitted.
func (c *CircuitBreaker) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if !c.openUntil.IsZero() && now.Before(c.openUntil) {
		return false
	}
	if !c.openUntil.IsZero() && now.After(c.openUntil) {
		c.failures = 0
		c.openUntil = time.Time{}
	}
	return true
}

// Success resets failure counts and closes the breaker.
func (c *CircuitBreaker) Success() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = 0
	c.openUntil = time.Time{}
}

// Failure records a failed call and opens the breaker when threshold is met.
func (c *CircuitBreaker) Failure() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if !c.openUntil.IsZero() && now.Before(c.openUntil) {
		return
	}
	if !c.openUntil.IsZero() && now.After(c.openUntil) {
		c.failures = 0
		c.openUntil = time.Time{}
	}

	c.failures++
	if c.failures >= c.threshold {
		c.openUntil = now.Add(c.resetAfter)
	}
}
