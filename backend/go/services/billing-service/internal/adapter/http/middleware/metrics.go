package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

var (
	metricsOnce         sync.Once
	metricsRegisterer   prometheus.Registerer = prometheus.DefaultRegisterer
	metricsRegistererMu sync.Mutex

	requestDuration *prometheus.HistogramVec
	requestTotal    *prometheus.CounterVec
	authDuration    *prometheus.HistogramVec
	authTotal       *prometheus.CounterVec
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
	if authDuration != nil {
		metricsRegisterer.Unregister(authDuration)
	}
	if authTotal != nil {
		metricsRegisterer.Unregister(authTotal)
	}

	metricsRegisterer = r
	metricsOnce = sync.Once{}
	requestDuration = nil
	requestTotal = nil
	authDuration = nil
	authTotal = nil
}

// MetricsRegisterer exposes the current registerer so other middleware can share it.
func MetricsRegisterer() prometheus.Registerer {
	return metricsRegisterer
}

// MetricsGatherer returns the gatherer associated with the current registerer (useful for tests).
func MetricsGatherer() prometheus.Gatherer {
	if g, ok := metricsRegisterer.(prometheus.Gatherer); ok {
		return g
	}
	return prometheus.DefaultGatherer
}

func initMetrics(service string) {
	metricsOnce.Do(func() {
		requestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "billing_service_request_duration_seconds",
				Help:    "Latency of HTTP requests",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status", "service"},
		)
		requestTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "billing_service_requests_total",
				Help: "Total HTTP requests",
			},
			[]string{"method", "path", "status", "service", "tenant_id"},
		)
		authDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "billing_service_auth_duration_seconds",
				Help:    "Latency of authentication decisions",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
			},
			[]string{"result", "service"},
		)
		authTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "billing_service_auth_total",
				Help: "Total authentication attempts",
			},
			[]string{"result", "service", "tenant_id"},
		)

		metricsRegisterer.MustRegister(requestDuration, requestTotal, authDuration, authTotal)
	})
}

// Metrics records request counts and latencies with path/method/status labels.
func Metrics(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		initMetrics(service)

		start := time.Now()
		c.Next()
		status := c.Writer.Status()

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		tenant := tenantLabel(c)
		statusLabel := httpStatusLabel(status)

		requestTotal.With(prometheus.Labels{
			"method":    c.Request.Method,
			"path":      path,
			"status":    statusLabel,
			"service":   service,
			"tenant_id": tenant,
		}).Inc()
		requestDuration.With(prometheus.Labels{
			"method":  c.Request.Method,
			"path":    path,
			"status":  statusLabel,
			"service": service,
		}).Observe(time.Since(start).Seconds())
	}
}

// AuthMetrics records auth outcomes/latency for routes that run platform-auth.
func AuthMetrics(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		initMetrics(service)
		start := time.Now()
		c.Next()

		result := authResultLabel(c.Writer.Status())
		tenant := tenantLabel(c)

		authTotal.With(prometheus.Labels{
			"result":    result,
			"service":   service,
			"tenant_id": tenant,
		}).Inc()
		authDuration.With(prometheus.Labels{
			"result":  result,
			"service": service,
		}).Observe(time.Since(start).Seconds())
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

func authResultLabel(code int) string {
	switch {
	case code == http.StatusUnauthorized:
		return "unauthorized"
	case code == http.StatusForbidden:
		return "forbidden"
	case code >= 500:
		return "error"
	default:
		return "ok"
	}
}

func tenantLabel(c *gin.Context) string {
	if t, err := authmw.GetTenantIDFromContext(c); err == nil && t != "" {
		return t
	}
	if h := c.GetHeader("X-Tenant-Id"); h != "" {
		return h
	}
	return "unknown"
}
