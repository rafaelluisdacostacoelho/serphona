package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics defines the observability surface used by the service.
type Metrics interface {
	RecordHTTPRequest(method, endpoint, tenant string, status int, duration time.Duration, requestSize, responseSize int64)
	RecordSessionCreated()
	RecordSessionEnded()
	RecordMessageProcessed()
	RecordLLMCall(model string, duration time.Duration, success bool)
	RecordTokens(model string, promptTokens, completionTokens int)
	RecordToolExecution(toolName string, success bool)
}

// PrometheusMetrics implements Metrics using Prometheus collectors.
type PrometheusMetrics struct {
	namespace  string
	service    string
	registerer prometheus.Registerer

	httpRequests  *prometheus.CounterVec
	httpDuration  *prometheus.HistogramVec
	sessionsMade  prometheus.Counter
	sessionsEnded prometheus.Counter
	msgProcessed  prometheus.Counter
	llmCalls      *prometheus.CounterVec
	llmLatency    *prometheus.HistogramVec
	tokensUsed    *prometheus.CounterVec
	toolExec      *prometheus.CounterVec
}

// NewPrometheusMetrics creates a metrics collector backed by the default Prometheus registerer.
func NewPrometheusMetrics(namespace string) Metrics {
	return NewPrometheusMetricsWithRegisterer(namespace, namespace, prometheus.DefaultRegisterer)
}

// NewPrometheusMetricsWithRegisterer allows overriding the registerer and service label.
func NewPrometheusMetricsWithRegisterer(namespace, service string, reg prometheus.Registerer) Metrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	m := &PrometheusMetrics{
		namespace:  namespace,
		service:    service,
		registerer: reg,
	}

	m.init()
	return m
}

func (m *PrometheusMetrics) init() {
	m.httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_http_requests_total", m.namespace),
		Help: "Total HTTP requests received.",
	}, []string{"method", "path", "status", "service", "tenant_id"})

	m.httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    fmt.Sprintf("%s_http_request_duration_seconds", m.namespace),
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status", "service"})

	m.sessionsMade = prometheus.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_sessions_created_total", m.namespace),
		Help: "Sessions created.",
	})

	m.sessionsEnded = prometheus.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_sessions_ended_total", m.namespace),
		Help: "Sessions ended.",
	})

	m.msgProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_messages_processed_total", m.namespace),
		Help: "Messages processed.",
	})

	m.llmCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_llm_calls_total", m.namespace),
		Help: "LLM calls by model and outcome.",
	}, []string{"model", "result", "service"})

	m.llmLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    fmt.Sprintf("%s_llm_call_duration_seconds", m.namespace),
		Help:    "LLM call latency in seconds.",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
	}, []string{"model", "service"})

	m.tokensUsed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_tokens_used_total", m.namespace),
		Help: "Tokens consumed by model and type.",
	}, []string{"model", "kind", "service"})

	m.toolExec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_tool_executions_total", m.namespace),
		Help: "Tool executions by tool and outcome.",
	}, []string{"tool", "result", "service"})

	m.registerer.MustRegister(
		m.httpRequests,
		m.httpDuration,
		m.sessionsMade,
		m.sessionsEnded,
		m.msgProcessed,
		m.llmCalls,
		m.llmLatency,
		m.tokensUsed,
		m.toolExec,
	)
}

// RecordHTTPRequest records HTTP request metrics with service and tenant labels.
func (m *PrometheusMetrics) RecordHTTPRequest(method, endpoint, tenant string, status int, duration time.Duration, requestSize, responseSize int64) {
	statusClass := statusBucket(status)
	m.httpRequests.WithLabelValues(method, endpoint, statusClass, m.service, tenant).Inc()
	m.httpDuration.WithLabelValues(method, endpoint, statusClass, m.service).Observe(duration.Seconds())
}

func (m *PrometheusMetrics) RecordSessionCreated() {
	m.sessionsMade.Inc()
}

func (m *PrometheusMetrics) RecordSessionEnded() {
	m.sessionsEnded.Inc()
}

func (m *PrometheusMetrics) RecordMessageProcessed() {
	m.msgProcessed.Inc()
}

func (m *PrometheusMetrics) RecordLLMCall(model string, duration time.Duration, success bool) {
	result := "success"
	if !success {
		result = "error"
	}
	m.llmCalls.WithLabelValues(model, result, m.service).Inc()
	m.llmLatency.WithLabelValues(model, m.service).Observe(duration.Seconds())
}

func (m *PrometheusMetrics) RecordTokens(model string, promptTokens, completionTokens int) {
	m.tokensUsed.WithLabelValues(model, "prompt", m.service).Add(float64(promptTokens))
	m.tokensUsed.WithLabelValues(model, "completion", m.service).Add(float64(completionTokens))
	m.tokensUsed.WithLabelValues(model, "total", m.service).Add(float64(promptTokens + completionTokens))
}

func (m *PrometheusMetrics) RecordToolExecution(toolName string, success bool) {
	result := "success"
	if !success {
		result = "error"
	}
	m.toolExec.WithLabelValues(toolName, result, m.service).Inc()
}

// NoOpMetrics is a no-op implementation for when metrics are disabled.
type NoOpMetrics struct{}

// NewNoOpMetrics creates a no-op metrics collector.
func NewNoOpMetrics() Metrics { return &NoOpMetrics{} }

func (m *NoOpMetrics) RecordHTTPRequest(method, endpoint, tenant string, status int, duration time.Duration, requestSize, responseSize int64) {
}
func (m *NoOpMetrics) RecordSessionCreated()                                            {}
func (m *NoOpMetrics) RecordSessionEnded()                                              {}
func (m *NoOpMetrics) RecordMessageProcessed()                                          {}
func (m *NoOpMetrics) RecordLLMCall(model string, duration time.Duration, success bool) {}
func (m *NoOpMetrics) RecordTokens(model string, promptTokens, completionTokens int)    {}
func (m *NoOpMetrics) RecordToolExecution(toolName string, success bool)                {}

func statusBucket(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	default:
		return "2xx"
	}
}
