package observability

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.uber.org/zap"
)

var (
	auditOnce            sync.Once
	auditCounter         *prometheus.CounterVec
	authLatency          *prometheus.HistogramVec
	tenantMissingCounter *prometheus.CounterVec
)

func initAudit() {
	auditOnce.Do(func() {
		auditCounter = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_gateway_auth_events_total",
				Help: "Auth events by type and outcome",
			},
			[]string{"event", "outcome"},
		)
		authLatency = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "auth_gateway_auth_duration_seconds",
				Help:    "Duration of auth flows by event and outcome",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"event", "outcome"},
		)
		tenantMissingCounter = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_gateway_tenant_missing_total",
				Help: "Requests where tenant_id was missing or empty",
			},
			[]string{"path"},
		)
		prometheus.MustRegister(auditCounter, authLatency, tenantMissingCounter)
	})
}

// RecordAuthEvent logs a structured auth event and increments the corresponding counter.
func RecordAuthEvent(ctx context.Context, logger *zap.Logger, event, outcome string, details map[string]any) {
	if logger == nil {
		logger = zap.NewNop()
	}

	if event == "" {
		event = "unknown"
	}
	if outcome == "" {
		outcome = "unknown"
	}

	initAudit()
	auditCounter.With(prometheus.Labels{"event": event, "outcome": outcome}).Inc()

	fields := []zap.Field{
		zap.String("event", event),
		zap.String("outcome", outcome),
	}

	if reqID, err := authmw.RequestIDFromContext(ctx); err == nil && reqID != "" {
		fields = append(fields, zap.String("request_id", reqID))
	}

	for k, v := range details {
		fields = append(fields, zap.Any(k, v))
	}

	logger.Info("auth_audit", fields...)
}

// ObserveAuthLatency records the duration of an auth flow.
func ObserveAuthLatency(event, outcome string, duration time.Duration) {
	initAudit()
	authLatency.With(prometheus.Labels{"event": eventLabel(event), "outcome": outcomeLabel(outcome)}).Observe(duration.Seconds())
}

// RecordTenantMissing counts requests missing tenant context.
func RecordTenantMissing(path string) {
	initAudit()
	tenantMissingCounter.With(prometheus.Labels{"path": path}).Inc()
}

func eventLabel(event string) string {
	if event == "" {
		return "unknown"
	}
	return event
}

func outcomeLabel(outcome string) string {
	if outcome == "" {
		return "unknown"
	}
	return outcome
}

// MaskEmail hides the local part of an email for logging/audit purposes.
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return email
	}
	local, domain := parts[0], parts[1]
	if len(local) <= 2 {
		return "***@" + domain
	}
	return local[:1] + "***" + local[len(local)-1:] + "@" + domain
}
