package invoke

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPrometheusSinkRecordsMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	sink := NewPrometheusSink(reg).(*promSink)

	labels := map[string]string{"tenant": "t1", "tool": "echo", "outcome": "ok"}
	sink.IncCounter("mcp_invocations_total", labels)
	sink.ObserveHistogram("mcp_invocation_latency_seconds", 0.5, labels)

	if got := testutil.ToFloat64(sink.counter.With(prometheus.Labels(labels))); got != 1 {
		t.Fatalf("expected counter=1, got %v", got)
	}
	hist, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	count := 0
	for _, mf := range hist {
		if mf.GetName() == "mcp_invocation_latency_seconds" {
			for _, m := range mf.Metric {
				count += int(m.GetHistogram().GetSampleCount())
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected histogram count=1, got %d", count)
	}
}
