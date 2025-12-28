package policy

import (
	"context"
	"strings"
	"sync"
	"time"
)

// Decision represents the result of evaluating a policy.
type Decision struct {
	Allowed     bool
	Reason      string
	MatchedRule string
	RateLimited bool
	RetryAfter  time.Duration
}

// Input captures the attributes evaluated by policies.
type Input struct {
	TenantID    string
	Tool        string
	AgentID     string
	Scopes      []string
	Environment string
}

// Evaluator evaluates allow/deny decisions for invocations.
type Evaluator interface {
	Evaluate(ctx context.Context, in Input) (Decision, error)
}

// Rule is a simple in-memory rule definition.
type Rule struct {
	ID                 string
	TenantID           string
	Tool               string
	Allow              bool
	RequireScopes      []string
	Environments       []string
	Priority           int // lower number means higher priority
	RateLimitPerMinute int // optional fixed window per tenant+tool+rule
}

// MemoryEvaluator applies a static ordered rule set (fail-closed by default).
type MemoryEvaluator struct {
	rules []Rule
	mu    sync.Mutex
	rate  map[string]*rateCounter // key: tenant|tool|rule
}

// NewMemoryEvaluator creates a new evaluator with ordered rules.
func NewMemoryEvaluator(rules []Rule) *MemoryEvaluator {
	return &MemoryEvaluator{rules: rules, rate: make(map[string]*rateCounter)}
}

// Evaluate applies rules in order of priority, then declaration order. Defaults to deny if none match.
func (m *MemoryEvaluator) Evaluate(_ context.Context, in Input) (Decision, error) {
	best := Rule{}
	matched := false
	for _, r := range m.rules {
		if !matchTenant(r.TenantID, in.TenantID) {
			continue
		}
		if !matchTool(r.Tool, in.Tool) {
			continue
		}
		if !matchEnv(r.Environments, in.Environment) {
			continue
		}
		if !hasAllScopes(in.Scopes, r.RequireScopes) {
			continue
		}
		if !matched || r.Priority < best.Priority {
			best = r
			matched = true
		}
	}

	if !matched {
		return Decision{Allowed: false, Reason: "no matching policy"}, nil
	}

	// Rate limiting per rule (fixed window per minute)
	if best.RateLimitPerMinute > 0 {
		if limited, retry := m.hitRateLimit(best, in); limited {
			return Decision{
				Allowed:     false,
				Reason:      "rate_limited",
				MatchedRule: best.ID,
				RateLimited: true,
				RetryAfter:  retry,
			}, nil
		}
	}

	return Decision{
		Allowed:     best.Allow,
		Reason:      ifThenElse(best.Allow, "allowed", "denied"),
		MatchedRule: best.ID,
	}, nil
}

type rateCounter struct {
	count     int
	resetTime time.Time
}

func (m *MemoryEvaluator) hitRateLimit(r Rule, in Input) (bool, time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := strings.ToLower(in.TenantID) + "|" + strings.ToLower(in.Tool) + "|" + r.ID
	c, ok := m.rate[key]
	now := time.Now()
	if !ok || now.After(c.resetTime) {
		m.rate[key] = &rateCounter{count: 1, resetTime: now.Add(time.Minute)}
		return false, 0
	}

	c.count++
	if c.count > r.RateLimitPerMinute {
		return true, time.Until(c.resetTime)
	}
	return false, 0
}

func matchTenant(ruleTenant, tenant string) bool {
	return ruleTenant == "" || strings.EqualFold(ruleTenant, tenant)
}

func matchTool(ruleTool, tool string) bool {
	return ruleTool == "" || strings.EqualFold(ruleTool, tool)
}

func matchEnv(envs []string, env string) bool {
	if len(envs) == 0 {
		return true
	}
	for _, e := range envs {
		if strings.EqualFold(e, env) {
			return true
		}
	}
	return false
}

func hasAllScopes(have, need []string) bool {
	if len(need) == 0 {
		return true
	}
	haveSet := make(map[string]struct{}, len(have))
	for _, s := range have {
		haveSet[strings.ToLower(s)] = struct{}{}
	}
	for _, n := range need {
		if _, ok := haveSet[strings.ToLower(n)]; !ok {
			return false
		}
	}
	return true
}

func ifThenElse(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
