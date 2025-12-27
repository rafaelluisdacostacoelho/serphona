package pgvector

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestPrometheusObserverCountsAndLatency(t *testing.T) {
	reg := prometheus.NewRegistry()
	obs, err := NewPrometheusObserver(PrometheusObserverConfig{Registerer: reg})
	if err != nil {
		t.Fatalf("new observer: %v", err)
	}

	ctx, end := obs.Trace(context.Background(), "pgvector.upsert")
	// simulate work
	time.Sleep(5 * time.Millisecond)
	end(nil)
	obs.RecordLatency(ctx, "pgvector.upsert", 5*time.Millisecond, nil)

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}

	var reqCount uint64
	var latencySamples uint64

	for _, mf := range mfs {
		switch mf.GetName() {
		case "platform_rag_pgvector_requests_total":
			for _, m := range mf.GetMetric() {
				if labelsMatch(m, map[string]string{"operation": "pgvector.upsert", "status": "ok"}) {
					reqCount = uint64(m.GetCounter().GetValue())
				}
			}
		case "platform_rag_pgvector_latency_seconds":
			for _, m := range mf.GetMetric() {
				if labelsMatch(m, map[string]string{"operation": "pgvector.upsert", "status": "ok"}) {
					latencySamples = m.GetHistogram().GetSampleCount()
				}
			}
		}
	}

	if reqCount != 1 {
		t.Fatalf("expected 1 ok request, got %d", reqCount)
	}
	if latencySamples != 2 { // Trace + RecordLatency
		t.Fatalf("expected 2 latency samples, got %d", latencySamples)
	}
}

func TestPrometheusObserverRegistersOnce(t *testing.T) {
	reg := prometheus.NewRegistry()
	_, err := NewPrometheusObserver(PrometheusObserverConfig{Registerer: reg})
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	// Second should re-use existing collectors, not fail.
	if _, err := NewPrometheusObserver(PrometheusObserverConfig{Registerer: reg}); err != nil {
		t.Fatalf("second register: %v", err)
	}
}

func labelsMatch(m *dto.Metric, expected map[string]string) bool {
	got := map[string]string{}
	for _, lp := range m.GetLabel() {
		got[lp.GetName()] = lp.GetValue()
	}
	if len(got) != len(expected) {
		return false
	}
	for k, v := range expected {
		if got[k] != v {
			return false
		}
	}
	return true
}

type flakyRegisterer struct{ calls int }

func (f *flakyRegisterer) Register(prometheus.Collector) error {
	f.calls++
	if f.calls == 2 {
		return errors.New("histogram register fail")
	}
	return nil
}

func (f *flakyRegisterer) MustRegister(...prometheus.Collector) {}
func (f *flakyRegisterer) Unregister(prometheus.Collector) bool { return true }

func TestPrometheusObserverHistogramRegisterError(t *testing.T) {
	reg := &flakyRegisterer{}
	if _, err := NewPrometheusObserver(PrometheusObserverConfig{Registerer: reg}); err == nil {
		t.Fatalf("expected histogram registration error")
	}
}
