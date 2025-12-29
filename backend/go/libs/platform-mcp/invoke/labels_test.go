package invoke

import (
	"context"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

func TestPolicyDecisionEnricher(t *testing.T) {
	ctx := WithPolicyDecision(context.Background(), "allow")
	labels := PolicyDecisionEnricher(ctx, protocol.InvocationRequest{}, protocol.InvocationEvent{}, nil)
	if labels == nil || labels["policy_decision"] != "allow" {
		t.Fatalf("expected policy_decision label, got %+v", labels)
	}
}

func TestPolicyDecisionEnricherMissing(t *testing.T) {
	ctx := context.Background()
	if labels := PolicyDecisionEnricher(ctx, protocol.InvocationRequest{}, protocol.InvocationEvent{}, nil); labels != nil {
		t.Fatalf("expected nil labels when decision missing, got %+v", labels)
	}
	ctx = WithPolicyDecision(ctx, "")
	if labels := PolicyDecisionEnricher(ctx, protocol.InvocationRequest{}, protocol.InvocationEvent{}, nil); labels != nil {
		t.Fatalf("expected nil labels when decision empty, got %+v", labels)
	}
}

func TestCacheHitEnricherMissing(t *testing.T) {
	if labels := CacheHitEnricher(context.Background(), protocol.InvocationRequest{}, protocol.InvocationEvent{}, nil); labels != nil {
		t.Fatalf("expected nil labels when cache hit unset, got %+v", labels)
	}
}
