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

func TestMemoryEvaluatorMatrix(t *testing.T) {
	ev := NewMemoryEvaluator([]Rule{
		{ID: "deny-prod", TenantID: "t1", Tool: "echo", Allow: false, Environments: []string{"prod"}, Priority: 1},
		{ID: "allow-scope", TenantID: "t1", Tool: "echo", Allow: true, RequireScopes: []string{"tool:run"}, Priority: 5},
		{ID: "fallback", TenantID: "t1", Allow: true, Priority: 100},
	})

	tests := []struct {
		name    string
		in      Input
		allowed bool
		rule    string
		reason  string
	}{
		{
			name:    "prod denied despite scope",
			in:      Input{TenantID: "t1", Tool: "echo", Environment: "prod", Scopes: []string{"tool:run"}},
			allowed: false,
			rule:    "deny-prod",
			reason:  "denied",
		},
		{
			name:    "dev requires scope",
			in:      Input{TenantID: "t1", Tool: "echo", Environment: "dev", Scopes: []string{"tool:run"}},
			allowed: true,
			rule:    "allow-scope",
			reason:  "allowed",
		},
		{
			name:    "dev missing scope falls back",
			in:      Input{TenantID: "t1", Tool: "echo", Environment: "dev", Scopes: []string{"other"}},
			allowed: true,
			rule:    "fallback",
			reason:  "allowed",
		},
		{
			name:    "tenant mismatch deny",
			in:      Input{TenantID: "t2", Tool: "echo", Environment: "dev", Scopes: []string{"tool:run"}},
			allowed: false,
			rule:    "",
			reason:  "no matching policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec, err := ev.Evaluate(context.Background(), tt.in)
			if err != nil {
				t.Fatalf("evaluate err: %v", err)
			}
			if dec.Allowed != tt.allowed {
				t.Fatalf("allowed mismatch: got %v want %v", dec.Allowed, tt.allowed)
			}
			if dec.MatchedRule != tt.rule {
				t.Fatalf("rule mismatch: got %s want %s", dec.MatchedRule, tt.rule)
			}
			if dec.Reason != tt.reason {
				t.Fatalf("reason mismatch: got %s want %s", dec.Reason, tt.reason)
			}
		})
	}
}
