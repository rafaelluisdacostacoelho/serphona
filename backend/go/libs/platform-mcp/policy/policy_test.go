package policy

import (
	"context"
	"testing"
)

func TestMemoryEvaluator(t *testing.T) {
	ev := NewMemoryEvaluator([]Rule{
		{ID: "deny-env", TenantID: "t1", Tool: "echo", Allow: false, Environments: []string{"prod"}, Priority: 1},
		{ID: "allow-scope", TenantID: "t1", Tool: "echo", Allow: true, RequireScopes: []string{"tool:run"}, Priority: 10},
	})

	in := Input{TenantID: "t1", Tool: "echo", Environment: "prod", Scopes: []string{"tool:run"}}
	dec, err := ev.Evaluate(context.Background(), in)
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if dec.Allowed {
		t.Fatalf("expected deny due to higher priority rule")
	}
	if dec.MatchedRule != "deny-env" {
		t.Fatalf("expected matched rule deny-env, got %s", dec.MatchedRule)
	}

	in.Environment = "dev"
	dec, err = ev.Evaluate(context.Background(), in)
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if !dec.Allowed {
		t.Fatalf("expected allow in dev")
	}
	if dec.MatchedRule != "allow-scope" {
		t.Fatalf("expected allow-scope matched, got %s", dec.MatchedRule)
	}
}

func TestMemoryEvaluatorRateLimit(t *testing.T) {
	ev := NewMemoryEvaluator([]Rule{
		{ID: "rate", TenantID: "t1", Tool: "echo", Allow: true, Priority: 1, RateLimitPerMinute: 1},
	})

	in := Input{TenantID: "t1", Tool: "echo"}
	dec, err := ev.Evaluate(context.Background(), in)
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if !dec.Allowed {
		t.Fatalf("first call should be allowed")
	}

	dec, err = ev.Evaluate(context.Background(), in)
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if dec.Allowed || !dec.RateLimited {
		t.Fatalf("second call should be rate limited")
	}
}
