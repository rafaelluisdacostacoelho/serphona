package middleware

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestSetAuthMetricsServiceAndGatherer(t *testing.T) {
	reg := prometheus.NewRegistry()
	SetMetricsRegisterer(reg)
	t.Cleanup(func() {
		SetMetricsRegisterer(nil)
		SetAuthMetricsService("")
	})

	SetAuthMetricsService("svc-a")
	observeAuth("http", "ok", "tenant-1", time.Now())

	mfs, err := MetricsGatherer().Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	foundService := false
	for _, mf := range mfs {
		if mf.GetName() != MetricAuthRequestsTotal {
			continue
		}
		for _, m := range mf.Metric {
			labels := map[string]string{}
			for _, lp := range m.Label {
				labels[lp.GetName()] = lp.GetValue()
			}
			if labels["service"] == "svc-a" {
				foundService = true
			}
		}
	}
	if !foundService {
		t.Fatalf("expected metric with service label svc-a")
	}

	SetAuthMetricsService("")
	observeAuth("http", "error", "tenant-2", time.Now())

	mfs, err = MetricsGatherer().Gather()
	if err != nil {
		t.Fatalf("gather metrics after reset: %v", err)
	}

	sawUnknown := false
	for _, mf := range mfs {
		if mf.GetName() != MetricAuthRequestsTotal {
			continue
		}
		for _, m := range mf.Metric {
			labels := map[string]string{}
			for _, lp := range m.Label {
				labels[lp.GetName()] = lp.GetValue()
			}
			if labels["service"] == "unknown" {
				sawUnknown = true
			}
		}
	}
	if !sawUnknown {
		t.Fatalf("expected metric with service label unknown after reset")
	}
}

func TestDefaultServiceNameEnv(t *testing.T) {
	t.Setenv("SERVICE_NAME", "svc-env")
	if got := defaultServiceName(); got != "svc-env" {
		t.Fatalf("expected env service name, got %s", got)
	}

	t.Setenv("SERVICE_NAME", "")
	if got := defaultServiceName(); got != "unknown" {
		t.Fatalf("expected fallback unknown, got %s", got)
	}
}
