package invoke

import (
	"context"
	"sync"
	"time"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// RateLimiter decides if a tenant/tool invocation is allowed.
type RateLimiter interface {
	Allow(ctx context.Context, tenantID, tool string) bool
}

// RateLimitExecutor rejects requests when the limiter denies.
type RateLimitExecutor struct {
	inner    Executor
	limiter  RateLimiter
	clock    func() time.Time
	cooldown time.Duration
}

// NewRateLimitExecutor wraps an executor with a rate limiter; cooldown adds optional sleep before deny (0 disables).
func NewRateLimitExecutor(inner Executor, limiter RateLimiter, cooldown time.Duration) *RateLimitExecutor {
	return &RateLimitExecutor{inner: inner, limiter: limiter, clock: time.Now, cooldown: cooldown}
}

// Invoke enforces rate limit before delegating.
func (r *RateLimitExecutor) Invoke(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	if r.limiter != nil && !r.limiter.Allow(ctx, req.TenantID, req.Tool.Name) {
		if r.cooldown > 0 {
			select {
			case <-ctx.Done():
			default:
				t := time.NewTimer(r.cooldown)
				select {
				case <-t.C:
				case <-ctx.Done():
				}
				t.Stop()
			}
		}
		return nil, mcperrors.New(mcperrors.ErrRateLimited, "rate limited")
	}
	return r.inner.Invoke(ctx, req)
}

// MemoryRateLimiter is a simple token bucket per (tenant,tool).
type MemoryRateLimiter struct {
	mu    sync.Mutex
	bkts  map[string]*bucket
	rate  float64 // tokens per second
	burst float64
	clock func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewMemoryRateLimiter creates a per-key token bucket limiter.
func NewMemoryRateLimiter(rate float64, burst int) *MemoryRateLimiter {
	if rate <= 0 {
		rate = 1
	}
	if burst <= 0 {
		burst = 1
	}
	return &MemoryRateLimiter{bkts: make(map[string]*bucket), rate: rate, burst: float64(burst), clock: time.Now}
}

// Allow consumes a token if available.
func (m *MemoryRateLimiter) Allow(_ context.Context, tenantID, tool string) bool {
	key := tenantID + ":" + tool
	m.mu.Lock()
	b, ok := m.bkts[key]
	now := m.clock()
	if !ok {
		b = &bucket{tokens: m.burst - 1, last: now}
		m.bkts[key] = b
		m.mu.Unlock()
		return true
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(m.burst, b.tokens+elapsed*m.rate)
	b.last = now
	if b.tokens < 1 {
		m.mu.Unlock()
		return false
	}
	b.tokens -= 1
	m.mu.Unlock()
	return true
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
