package invoke

import "github.com/prometheus/client_golang/prometheus"

// promSink implements MetricsSink backed by Prometheus client_golang.
type promSink struct {
	counter *prometheus.CounterVec
	hist    *prometheus.HistogramVec
}

// NewPrometheusSink registers and returns a MetricsSink using the provided registerer.
// Labels: tenant, tool, outcome.
func NewPrometheusSink(reg prometheus.Registerer) MetricsSink {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mcp_invocations_total",
		Help: "Total MCP invocations by outcome",
	}, []string{"tenant", "tool", "outcome"})
	hist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mcp_invocation_latency_seconds",
		Help:    "MCP invocation latency seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"tenant", "tool", "outcome"})
	reg.MustRegister(counter, hist)
	return &promSink{counter: counter, hist: hist}
}

func (p *promSink) IncCounter(_ string, labels map[string]string) {
	p.counter.With(prometheus.Labels(labels)).Inc()
}

func (p *promSink) ObserveHistogram(_ string, value float64, labels map[string]string) {
	p.hist.With(prometheus.Labels(labels)).Observe(value)
}
