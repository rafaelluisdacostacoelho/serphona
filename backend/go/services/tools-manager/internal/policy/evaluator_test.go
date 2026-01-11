package policy

import "testing"

func TestEvaluatorPrefersHigherWeight(t *testing.T) {
	rules := []Rule{
		{Effect: DecisionAllow, TenantID: "t1", Weight: 1},
		{Effect: DecisionDeny, TenantID: "t1", Weight: 2},
	}
	eval := NewEvaluator(rules)
	if got := eval.Evaluate(Request{TenantID: "t1"}); got != DecisionDeny {
		t.Fatalf("expected deny, got %s", got)
	}
}

func TestEvaluatorDenyBeatsAllowOnTie(t *testing.T) {
	rules := []Rule{
		{Effect: DecisionAllow, TenantID: "t1", Weight: 5},
		{Effect: DecisionDeny, TenantID: "t1", Weight: 5},
	}
	eval := NewEvaluator(rules)
	if got := eval.Evaluate(Request{TenantID: "t1"}); got != DecisionDeny {
		t.Fatalf("expected deny on tie, got %s", got)
	}
}

func TestEvaluatorMatchesScopesAndRoles(t *testing.T) {
	rules := []Rule{
		{Effect: DecisionAllow, TenantID: "t1", Scopes: []string{"write"}, Roles: []string{"admin"}},
	}
	eval := NewEvaluator(rules)

	if got := eval.Evaluate(Request{TenantID: "t1", Scopes: []string{"write"}, Roles: []string{"admin"}}); got != DecisionAllow {
		t.Fatalf("expected allow with matching scope/role, got %s", got)
	}
	if got := eval.Evaluate(Request{TenantID: "t1", Scopes: []string{"read"}, Roles: []string{"admin"}}); got != DecisionAllow {
		t.Fatalf("scopes are only required in rule; non-required should still allow, got %s", got)
	}
	if got := eval.Evaluate(Request{TenantID: "t1", Scopes: []string{"write"}, Roles: []string{"user"}}); got != DecisionAllow {
		t.Fatalf("roles optional; only deny when rule roles are unmet, got %s", got)
	}

	denyRule := Rule{Effect: DecisionDeny, TenantID: "t1", Scopes: []string{"write"}, Roles: []string{"admin"}, Weight: 10}
	eval = NewEvaluator([]Rule{denyRule})
	if got := eval.Evaluate(Request{TenantID: "t1", Scopes: []string{"write"}, Roles: []string{"admin"}}); got != DecisionDeny {
		t.Fatalf("expected deny when rule matches, got %s", got)
	}
}

func TestEvaluatorFallbackAllow(t *testing.T) {
	eval := NewEvaluator(nil)
	if got := eval.Evaluate(Request{TenantID: "no-rules"}); got != DecisionAllow {
		t.Fatalf("expected fallback allow, got %s", got)
	}
}
