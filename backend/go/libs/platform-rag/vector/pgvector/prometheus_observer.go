package pgvector

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusObserver emits counters and latencies for pgvector store operations.
// Labels: operation (e.g., pgvector.upsert/query/ping), status (ok|error).
// Plug it into Config.Observer to capture metrics.
type PrometheusObserver struct {
	requests *prometheus.CounterVec
	latency  *prometheus.HistogramVec
}

// PrometheusObserverConfig allows customizing metric names/help/buckets.
type PrometheusObserverConfig struct {
	// Registerer defaults to prometheus.DefaultRegisterer when nil.
	Registerer prometheus.Registerer
	// RequestsCounterOpts allows overriding counter name/help; labels are fixed.
	RequestsCounterOpts prometheus.CounterOpts
	// LatencyHistogramOpts allows overriding histogram options/buckets.
	LatencyHistogramOpts prometheus.HistogramOpts
}

// NewPrometheusObserver builds a Prometheus-backed observer and registers metrics.
func NewPrometheusObserver(cfg PrometheusObserverConfig) (*PrometheusObserver, error) {
	reg := cfg.Registerer
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	counterOpts := cfg.RequestsCounterOpts
	if counterOpts.Name == "" {
		counterOpts.Name = "platform_rag_pgvector_requests_total"
	}
	if counterOpts.Help == "" {
		counterOpts.Help = "Count of pgvector store operations by status"
	}

	histOpts := cfg.LatencyHistogramOpts
	if histOpts.Name == "" {
		histOpts.Name = "platform_rag_pgvector_latency_seconds"
	}
	if histOpts.Help == "" {
		histOpts.Help = "Latency of pgvector store operations"
	}
	if histOpts.Buckets == nil {
		histOpts.Buckets = prometheus.DefBuckets
	}

	requests := prometheus.NewCounterVec(counterOpts, []string{"operation", "status"})
	latency := prometheus.NewHistogramVec(histOpts, []string{"operation", "status"})

	if err := reg.Register(requests); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			requests = are.ExistingCollector.(*prometheus.CounterVec)
		} else {
			return nil, err
		}
	}
	if err := reg.Register(latency); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			latency = are.ExistingCollector.(*prometheus.HistogramVec)
		} else {
			return nil, err
		}
	}

	return &PrometheusObserver{requests: requests, latency: latency}, nil
}

// Trace increments the request counter at the end of an operation.
func (p *PrometheusObserver) Trace(ctx context.Context, operation string) (context.Context, func(error)) {
	start := time.Now()
	return ctx, func(err error) {
		status := statusLabel(err)
		p.requests.WithLabelValues(operation, status).Inc()
		p.latency.WithLabelValues(operation, status).Observe(time.Since(start).Seconds())
	}
}

// RecordLatency records latency (noop when nil observer).
func (p *PrometheusObserver) RecordLatency(_ context.Context, operation string, d time.Duration, err error) {
	status := statusLabel(err)
	p.latency.WithLabelValues(operation, status).Observe(d.Seconds())
}

func statusLabel(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}
