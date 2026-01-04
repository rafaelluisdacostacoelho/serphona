package middleware

import (
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce         sync.Once
	metricsRegisterer   prometheus.Registerer = prometheus.DefaultRegisterer
	metricsRegistererMu sync.Mutex

	authRequests *prometheus.CounterVec
	authLatency  *prometheus.HistogramVec
	authService  = defaultServiceName()
)

const (
	MetricAuthRequestsTotal          = "auth_requests_total"
	MetricAuthRequestDurationSeconds = "auth_request_duration_seconds"
)

// SetMetricsRegisterer allows overriding the Prometheus registerer (useful for tests).
func SetMetricsRegisterer(r prometheus.Registerer) {
	metricsRegistererMu.Lock()
	defer metricsRegistererMu.Unlock()

	if r == nil {
		r = prometheus.DefaultRegisterer
	}

	// Unregister existing collectors from the current registerer to avoid duplicate registration when swapping.
	if authRequests != nil {
		metricsRegisterer.Unregister(authRequests)
	}
	if authLatency != nil {
		metricsRegisterer.Unregister(authLatency)
	}

	metricsRegisterer = r
	metricsOnce = sync.Once{}
	authRequests = nil
	authLatency = nil
}

// SetAuthMetricsService overrides the service label used in auth metrics.
func SetAuthMetricsService(service string) {
	if service == "" {
		authService = "unknown"
		return
	}
	authService = service
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
		authRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: MetricAuthRequestsTotal,
			Help: "Count of authentication middleware decisions.",
		}, []string{"transport", "result", "service", "tenant_id", "component"})

		authLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    MetricAuthRequestDurationSeconds,
			Help:    "Latency of authentication middleware decisions.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		}, []string{"transport", "result", "service", "component"})

		metricsRegisterer.MustRegister(authRequests, authLatency)
	})
}

func recordAuthSuccess(transport, tenant string, start time.Time) {
	observeAuth(transport, "ok", tenant, start)
}

func recordAuthError(transport string, mapped authMappedError, tenant string, start time.Time) {
	result := "unauthorized"
	switch mapped.status {
	case 403:
		result = "forbidden"
	default:
		if mapped.status >= 500 {
			result = "error"
		}
	}
	observeAuth(transport, result, tenant, start)
}

func observeAuth(transport, result, tenant string, start time.Time) {
	initMetrics()
	elapsed := time.Since(start).Seconds()
	component := transport
	authRequests.WithLabelValues(transport, result, authService, tenant, component).Inc()
	authLatency.WithLabelValues(transport, result, authService, component).Observe(elapsed)
}

func defaultServiceName() string {
	if v := os.Getenv("SERVICE_NAME"); v != "" {
		return v
	}
	return "unknown"
}
