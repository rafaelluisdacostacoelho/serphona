package embedding

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Client computes embeddings for given inputs.
type Client interface {
	Embed(ctx context.Context, inputs []string) ([][]float32, error)
}

// RetryableClient decorates a Client with retries/backoff for transient failures.
type RetryableClient struct {
	Inner      Client
	MaxRetries int
	BackoffMin time.Duration
	BackoffMax time.Duration
}

// Embed proxies to the inner client with simple exponential backoff.
func (r RetryableClient) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if r.Inner == nil {
		return nil, ErrNoInnerClient
	}
	if r.MaxRetries <= 0 {
		r.MaxRetries = 3
	}
	if r.BackoffMin <= 0 {
		r.BackoffMin = 100 * time.Millisecond
	}
	if r.BackoffMax <= 0 {
		r.BackoffMax = 2 * time.Second
	}

	backoff := r.BackoffMin
	var lastErr error
	for attempt := 0; attempt < r.MaxRetries; attempt++ {
		res, err := r.Inner.Embed(ctx, inputs)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if attempt == r.MaxRetries-1 {
			break
		}
		t := backoff
		backoff *= 2
		if backoff > r.BackoffMax {
			backoff = r.BackoffMax
		}
		time.Sleep(t)
	}
	return nil, lastErr
}

// ErrNoInnerClient indicates the decorator has no inner client set.
var ErrNoInnerClient = errors.New("retryable client missing inner client")

// ErrQuotaExceeded indicates embedding was rejected due to quota limits.
var ErrQuotaExceeded = errors.New("embedding quota exceeded")

// quotaCtxKey stores tenant+namespace on context for quota enforcement.
type quotaCtxKey struct{}

// WithEmbeddingQuotaScope attaches tenant and namespace to context.
func WithEmbeddingQuotaScope(ctx context.Context, tenantID, namespace string) context.Context {
	return context.WithValue(ctx, quotaCtxKey{}, struct{ tenant, ns string }{tenant: tenantID, ns: namespace})
}

func quotaScopeFromContext(ctx context.Context) (tenant string, ns string) {
	if v, ok := ctx.Value(quotaCtxKey{}).(struct{ tenant, ns string }); ok {
		return v.tenant, v.ns
	}
	return "unknown", "unknown"
}

// QuotaConfig configures token/day and TPS limits plus cost metadata.
type QuotaConfig struct {
	TokensPerDay int
	TPS          int
	CostPer1KUSD float64
	Now          func() time.Time
}

type quotaState struct {
	day    int
	tokens int
	calls  []time.Time
}

// QuotaClient enforces per-tenant quotas before delegating to an inner Client.
type QuotaClient struct {
	Inner  Client
	Config QuotaConfig

	mu     sync.Mutex
	states map[string]*quotaState
}

// Embed estimates tokens, enforces quotas (daily tokens and TPS), then proxies to inner.
func (q *QuotaClient) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if q.Inner == nil {
		return nil, ErrNoInnerClient
	}
	cfg := q.Config
	if cfg.TokensPerDay <= 0 {
		cfg.TokensPerDay = 200000
	}
	if cfg.TPS <= 0 {
		cfg.TPS = 5
	}
	if cfg.CostPer1KUSD <= 0 {
		cfg.CostPer1KUSD = 0.00013
	}
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	tenant, ns := quotaScopeFromContext(ctx)
	tokens := estimateTokens(inputs)
	if err := q.enforceQuota(tenant, ns, tokens, nowFn()); err != nil {
		return nil, err
	}
	return q.Inner.Embed(ctx, inputs)
}

func (q *QuotaClient) enforceQuota(tenant, ns string, tokens int, now time.Time) error {
	key := tenant + "::" + ns
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.states == nil {
		q.states = make(map[string]*quotaState)
	}
	st, ok := q.states[key]
	if !ok {
		st = &quotaState{day: dayOrdinal(now)}
		q.states[key] = st
	}
	today := dayOrdinal(now)
	if st.day != today {
		st.day = today
		st.tokens = 0
		st.calls = st.calls[:0]
	}
	if st.tokens+tokens > q.Config.TokensPerDay && q.Config.TokensPerDay > 0 {
		return ErrQuotaExceeded
	}
	// TPS window of 1 second
	cutoff := now.Add(-1 * time.Second)
	filtered := st.calls[:0]
	for _, t := range st.calls {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	st.calls = append(filtered, now)
	if len(st.calls) > q.Config.TPS && q.Config.TPS > 0 {
		// revert call record to avoid drift
		st.calls = st.calls[:len(st.calls)-1]
		return ErrQuotaExceeded
	}

	st.tokens += tokens
	return nil
}

func estimateTokens(inputs []string) int {
	total := 0
	for _, s := range inputs {
		total += len(s)
	}
	tokens := total / 4
	if tokens < 1 {
		tokens = 1
	}
	return tokens
}

func dayOrdinal(t time.Time) int {
	return t.UTC().YearDay() + t.UTC().Year()*400 // simple monotonic ordinal
}
