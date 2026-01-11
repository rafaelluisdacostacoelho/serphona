package quota

import (
	"testing"
	"time"
)

func TestLimiterAllowsWhenNoRule(t *testing.T) {
	limiter := NewLimiter()
	decision := limiter.Evaluate(time.Now(), Request{TenantID: "t1"}, nil)
	if !decision.Allowed {
		t.Fatalf("expected allow when no rule, got deny")
	}
}

func TestLimiterEnforcesMinuteLimit(t *testing.T) {
	limiter := NewLimiter()
	now := time.Now()
	limit := 2
	rule := Rule{TenantID: "t1", LimitPerMinute: &limit}

	for i := 0; i < 2; i++ {
		d := limiter.Evaluate(now, Request{TenantID: "t1"}, []Rule{rule})
		if !d.Allowed {
			t.Fatalf("request %d unexpectedly denied", i)
		}
	}
	d := limiter.Evaluate(now, Request{TenantID: "t1"}, []Rule{rule})
	if d.Allowed {
		t.Fatalf("expected deny on exceeding minute limit")
	}
}

func TestLimiterResetsAfterWindow(t *testing.T) {
	limiter := NewLimiter()
	now := time.Now()
	limit := 1
	rule := Rule{TenantID: "t1", LimitPerMinute: &limit}

	d1 := limiter.Evaluate(now, Request{TenantID: "t1"}, []Rule{rule})
	if !d1.Allowed {
		t.Fatalf("first request should be allowed")
	}

	later := now.Add(time.Minute + time.Second)
	d2 := limiter.Evaluate(later, Request{TenantID: "t1"}, []Rule{rule})
	if !d2.Allowed {
		t.Fatalf("expected allowance after window reset")
	}
}

func TestLimiterMatchesSpecificRule(t *testing.T) {
	limiter := NewLimiter()
	limit := 1
	rules := []Rule{
		{TenantID: "t1", AgentID: "a1", LimitPerMinute: &limit},
		{TenantID: "t1", LimitPerMinute: nil},
	}
	now := time.Now()

	d := limiter.Evaluate(now, Request{TenantID: "t1", AgentID: "a1"}, rules)
	if !d.Allowed {
		t.Fatalf("expected allow on specific rule")
	}
	d = limiter.Evaluate(now, Request{TenantID: "t1", AgentID: "a1"}, rules)
	if d.Allowed {
		t.Fatalf("expected deny after specific rule quota exceeded")
	}
}
