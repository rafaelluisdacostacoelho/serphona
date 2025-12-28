package policy

import (
	"context"
	"errors"
	"time"
)

// MetricsSink abstracts metrics backends for policy evaluation.
type MetricsSink interface {
	IncCounter(name string, labels map[string]string)
	ObserveHistogram(name string, value float64, labels map[string]string)
}

// MetricsEvaluator wraps an Evaluator to emit decision metrics.
type MetricsEvaluator struct {
	inner Evaluator
	sink  MetricsSink
}

// NewMetricsEvaluator creates a metrics-emitting evaluator.
func NewMetricsEvaluator(inner Evaluator, sink MetricsSink) *MetricsEvaluator {
	return &MetricsEvaluator{inner: inner, sink: sink}
}

// Evaluate delegates to the inner evaluator and records metrics by tenant/tool/decision.
func (m *MetricsEvaluator) Evaluate(ctx context.Context, in Input) (Decision, error) {
	if m == nil {
		return Decision{}, errors.New("metrics evaluator is nil")
	}
	if m.inner == nil {
		return Decision{}, errors.New("inner evaluator is nil")
	}
	if m.sink == nil {
		return m.inner.Evaluate(ctx, in)
	}

	start := time.Now()
	dec, err := m.inner.Evaluate(ctx, in)
	elapsed := time.Since(start)

	decision := "deny"
	if err != nil {
		decision = "error"
	} else if dec.Allowed {
		decision = "allow"
	}

	labels := map[string]string{
		"tenant":      in.TenantID,
		"tool":        in.Tool,
		"environment": in.Environment,
		"decision":    decision,
	}
	if dec.MatchedRule != "" {
		labels["rule"] = dec.MatchedRule
	}
	if dec.RateLimited {
		labels["rate_limited"] = "true"
	} else {
		labels["rate_limited"] = "false"
	}

	m.sink.IncCounter("mcp_policy_evaluations_total", labels)
	m.sink.ObserveHistogram("mcp_policy_evaluation_latency_seconds", elapsed.Seconds(), labels)

	return dec, err
}
