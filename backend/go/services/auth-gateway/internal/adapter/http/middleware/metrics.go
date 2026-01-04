package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce         sync.Once
	metricsRegisterer   prometheus.Registerer = prometheus.DefaultRegisterer
	metricsRegistererMu sync.Mutex

	requestDuration *prometheus.HistogramVec
	requestTotal    *prometheus.CounterVec
)

// SetMetricsRegisterer allows overriding the Prometheus registerer (useful for tests).
func SetMetricsRegisterer(r prometheus.Registerer) {
	metricsRegistererMu.Lock()
	defer metricsRegistererMu.Unlock()

	if r == nil {
		r = prometheus.DefaultRegisterer
	}

	if requestDuration != nil {
		metricsRegisterer.Unregister(requestDuration)
	}
	if requestTotal != nil {
		metricsRegisterer.Unregister(requestTotal)
	}

	metricsRegisterer = r
	metricsOnce = sync.Once{}
	requestDuration = nil
	requestTotal = nil
}

// MetricsGatherer returns the gatherer associated with the current registerer (useful for tests).
func MetricsGatherer() prometheus.Gatherer {
	if g, ok := metricsRegisterer.(prometheus.Gatherer); ok {
		return g
	}
	return prometheus.DefaultGatherer
}

func initMetrics() {
	metricsOnce.Do(func() {
		requestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "auth_gateway_request_duration_seconds",
				Help:    "Latency of HTTP requests",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		)
		requestTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_gateway_requests_total",
				Help: "Total HTTP requests",
			},
			[]string{"method", "path", "status"},
		)

		metricsRegisterer.MustRegister(requestDuration, requestTotal)
	})
}

// Metrics records request counts and latencies with path/method/status labels.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		initMetrics()

		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		labels := prometheus.Labels{
			"method": c.Request.Method,
			"path":   path,
			"status": httpStatusLabel(status),
		}
		requestTotal.With(labels).Inc()
		requestDuration.With(labels).Observe(time.Since(start).Seconds())
	}
}

func httpStatusLabel(code int) string {
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
