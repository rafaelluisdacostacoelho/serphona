package invoke

import (
	"context"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type cacheHitKey struct{}
type policyDecisionKey struct{}

// WithCacheHit stores cache_hit label value in context for metrics enrichers.
func WithCacheHit(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, cacheHitKey{}, value)
}

// WithPolicyDecision stores policy_decision label value in context for metrics enrichers.
func WithPolicyDecision(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, policyDecisionKey{}, value)
}

// CacheHitEnricher reads cache_hit from context if present.
func CacheHitEnricher(ctx context.Context, _ protocol.InvocationRequest, _ protocol.InvocationEvent, _ error) map[string]string {
	if v, ok := ctx.Value(cacheHitKey{}).(string); ok && v != "" {
		return map[string]string{"cache_hit": v}
	}
	return nil
}

// PolicyDecisionEnricher reads policy_decision from context if present.
func PolicyDecisionEnricher(ctx context.Context, _ protocol.InvocationRequest, _ protocol.InvocationEvent, _ error) map[string]string {
	if v, ok := ctx.Value(policyDecisionKey{}).(string); ok && v != "" {
		return map[string]string{"policy_decision": v}
	}
	return nil
}
