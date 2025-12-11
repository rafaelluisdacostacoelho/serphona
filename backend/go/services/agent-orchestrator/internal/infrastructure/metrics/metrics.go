package metrics

import (
	"log"
	"time"
)

// Metrics interface defines methods for recording metrics
type Metrics interface {
	RecordHTTPRequest(method, endpoint string, status int, duration time.Duration, requestSize, responseSize int64)
	RecordSessionCreated()
	RecordSessionEnded()
	RecordMessageProcessed()
	RecordLLMCall(model string, duration time.Duration, success bool)
	RecordTokens(model string, promptTokens, completionTokens int)
	RecordToolExecution(toolName string, success bool)
}

// PrometheusMetrics implements Metrics using Prometheus (structure ready for integration)
type PrometheusMetrics struct {
	namespace string
	// When Prometheus library is added:
	// registry  *prometheus.Registry
	// collectors map[string]prometheus.Collector
}

// NewPrometheusMetrics creates a new Prometheus metrics collector
func NewPrometheusMetrics(namespace string) Metrics {
	m := &PrometheusMetrics{
		namespace: namespace,
	}

	// TODO: Initialize Prometheus collectors when library is added
	// Example:
	// m.registry = prometheus.NewRegistry()
	// m.httpRequestsTotal = promauto.With(m.registry).NewCounterVec(...)
	// m.httpRequestDuration = promauto.With(m.registry).NewHistogramVec(...)
	// etc.

	log.Printf("✅ Metrics collector initialized (namespace: %s)", namespace)
	return m
}

// RecordHTTPRequest records HTTP request metrics
func (m *PrometheusMetrics) RecordHTTPRequest(method, endpoint string, status int, duration time.Duration, requestSize, responseSize int64) {
	// TODO: Record to Prometheus when library is added
	// m.httpRequestsTotal.WithLabelValues(method, endpoint, statusClass(status)).Inc()
	// m.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())

	log.Printf("📊 HTTP: %s %s [%d] %.3fs req=%dB res=%dB",
		method, endpoint, status, duration.Seconds(), requestSize, responseSize)
}

// RecordSessionCreated records session creation
func (m *PrometheusMetrics) RecordSessionCreated() {
	// TODO: Record to Prometheus
	// m.sessionsCreated.Inc()
	log.Println("📊 Session created")
}

// RecordSessionEnded records session end
func (m *PrometheusMetrics) RecordSessionEnded() {
	// TODO: Record to Prometheus
	// m.sessionsEnded.Inc()
	log.Println("📊 Session ended")
}

// RecordMessageProcessed records message processing
func (m *PrometheusMetrics) RecordMessageProcessed() {
	// TODO: Record to Prometheus
	// m.messagesProcessed.Inc()
	log.Println("📊 Message processed")
}

// RecordLLMCall records LLM API call
func (m *PrometheusMetrics) RecordLLMCall(model string, duration time.Duration, success bool) {
	// TODO: Record to Prometheus
	// status := "success"
	// if !success { status = "error" }
	// m.llmCallsTotal.WithLabelValues(model, status).Inc()
	// m.llmCallDuration.WithLabelValues(model).Observe(duration.Seconds())

	status := "success"
	if !success {
		status = "error"
	}
	log.Printf("📊 LLM call: model=%s status=%s duration=%.3fs", model, status, duration.Seconds())
}

// RecordTokens records token usage
func (m *PrometheusMetrics) RecordTokens(model string, promptTokens, completionTokens int) {
	// TODO: Record to Prometheus
	// m.tokensUsed.WithLabelValues(model, "prompt").Add(float64(promptTokens))
	// m.tokensUsed.WithLabelValues(model, "completion").Add(float64(completionTokens))
	// m.tokensUsed.WithLabelValues(model, "total").Add(float64(promptTokens + completionTokens))

	log.Printf("📊 Tokens: model=%s prompt=%d completion=%d total=%d",
		model, promptTokens, completionTokens, promptTokens+completionTokens)
}

// RecordToolExecution records tool execution
func (m *PrometheusMetrics) RecordToolExecution(toolName string, success bool) {
	// TODO: Record to Prometheus
	// status := "success"
	// if !success { status = "error" }
	// m.toolExecutionsTotal.WithLabelValues(toolName, status).Inc()

	status := "success"
	if !success {
		status = "error"
	}
	log.Printf("📊 Tool execution: tool=%s status=%s", toolName, status)
}

// NoOpMetrics is a no-op implementation for when metrics are disabled
type NoOpMetrics struct{}

// NewNoOpMetrics creates a no-op metrics collector
func NewNoOpMetrics() Metrics {
	return &NoOpMetrics{}
}

func (m *NoOpMetrics) RecordHTTPRequest(method, endpoint string, status int, duration time.Duration, requestSize, responseSize int64) {
}

func (m *NoOpMetrics) RecordSessionCreated() {}

func (m *NoOpMetrics) RecordSessionEnded() {}

func (m *NoOpMetrics) RecordMessageProcessed() {}

func (m *NoOpMetrics) RecordLLMCall(model string, duration time.Duration, success bool) {}

func (m *NoOpMetrics) RecordTokens(model string, promptTokens, completionTokens int) {}

func (m *NoOpMetrics) RecordToolExecution(toolName string, success bool) {}
