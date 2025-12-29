package policy

import (
	"context"
	"errors"
	"testing"
)

type stubEvaluator struct {
	decision Decision
	err      error
}

func (s *stubEvaluator) Evaluate(context.Context, Input) (Decision, error) {
	return s.decision, s.err
}

type fakeSink struct {
	counters []metricCall
	hists    []histCall
}

type metricCall struct {
	name   string
	labels map[string]string
}

type histCall struct {
	name   string
	value  float64
	labels map[string]string
}

func (f *fakeSink) IncCounter(name string, labels map[string]string) {
	f.counters = append(f.counters, metricCall{name: name, labels: labels})
}

func (f *fakeSink) ObserveHistogram(name string, value float64, labels map[string]string) {
	f.hists = append(f.hists, histCall{name: name, value: value, labels: labels})
}

func TestMetricsEvaluatorRecordsDecision(t *testing.T) {
	sink := &fakeSink{}
	inner := &stubEvaluator{decision: Decision{Allowed: true, MatchedRule: "r1"}}
	ev := NewMetricsEvaluator(inner, sink)

	dec, err := ev.Evaluate(context.Background(), Input{TenantID: "t1", Tool: "echo", Environment: "prod"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dec.Allowed {
		t.Fatalf("expected allow decision")
	}

	if len(sink.counters) != 1 {
		t.Fatalf("expected 1 counter, got %d", len(sink.counters))
	}
	labels := sink.counters[0].labels
	if labels["decision"] != "allow" || labels["tenant"] != "t1" || labels["tool"] != "echo" || labels["environment"] != "prod" || labels["rule"] != "r1" || labels["rate_limited"] != "false" {
		t.Fatalf("unexpected labels: %#v", labels)
	}

	if len(sink.hists) != 1 {
		t.Fatalf("expected 1 histogram, got %d", len(sink.hists))
	}
	if sink.hists[0].name != "mcp_policy_evaluation_latency_seconds" {
		t.Fatalf("unexpected histogram name: %s", sink.hists[0].name)
	}
}

func TestMetricsEvaluatorErrorsAreLabeled(t *testing.T) {
	sink := &fakeSink{}
	inner := &stubEvaluator{err: errors.New("boom")}
	ev := NewMetricsEvaluator(inner, sink)

	_, err := ev.Evaluate(context.Background(), Input{TenantID: "t2", Tool: "plan"})
	if err == nil {
		t.Fatalf("expected error from evaluator")
	}

	if len(sink.counters) != 1 {
		t.Fatalf("expected 1 counter, got %d", len(sink.counters))
	}
	labels := sink.counters[0].labels
	if labels["decision"] != "error" || labels["tenant"] != "t2" || labels["tool"] != "plan" {
		t.Fatalf("unexpected labels: %#v", labels)
	}
	if labels["rate_limited"] != "false" {
		t.Fatalf("expected rate_limited false, got %s", labels["rate_limited"])
	}
}

func TestMetricsEvaluatorRateLimitedLabel(t *testing.T) {
	sink := &fakeSink{}
	inner := &stubEvaluator{decision: Decision{Allowed: false, MatchedRule: "r1", RateLimited: true}}
	ev := NewMetricsEvaluator(inner, sink)

	_, err := ev.Evaluate(context.Background(), Input{TenantID: "t3", Tool: "calc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sink.counters) != 1 || sink.counters[0].labels["rate_limited"] != "true" {
		t.Fatalf("expected rate_limited label true, got %+v", sink.counters)
	}
}

func TestMetricsEvaluatorNilReceiver(t *testing.T) {
	var ev *MetricsEvaluator
	if _, err := ev.Evaluate(context.Background(), Input{}); err == nil {
		t.Fatalf("expected error when evaluator is nil")
	}
}

func TestMetricsEvaluatorNilInner(t *testing.T) {
	ev := &MetricsEvaluator{sink: &fakeSink{}}
	if _, err := ev.Evaluate(context.Background(), Input{}); err == nil {
		t.Fatalf("expected error when inner evaluator nil")
	}
}

func TestMetricsEvaluatorNoSinkDelegates(t *testing.T) {
	inner := &stubEvaluator{decision: Decision{Allowed: true}}
	ev := NewMetricsEvaluator(inner, nil)
	dec, err := ev.Evaluate(context.Background(), Input{TenantID: "t1", Tool: "echo"})
	if err != nil || !dec.Allowed {
		t.Fatalf("expected delegated decision, got dec=%+v err=%v", dec, err)
	}
}
