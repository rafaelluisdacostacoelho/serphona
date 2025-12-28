package invoke

import (
	"context"
	"time"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// MetricsSink is an abstraction over metrics backends (Prometheus, OTEL, etc.).
type MetricsSink interface {
	IncCounter(name string, labels map[string]string)
	ObserveHistogram(name string, value float64, labels map[string]string)
}

// MetricsObserver emits basic invocation metrics via a MetricsSink.
// LabelEnricher can add/override labels for metrics.
type LabelEnricher func(ctx context.Context, req protocol.InvocationRequest, evt protocol.InvocationEvent, invokeErr error) map[string]string

type MetricsObserver struct {
	sink      MetricsSink
	enrichers []LabelEnricher
}

// NewMetricsObserver builds an observer with optional label enrichers.
func NewMetricsObserver(sink MetricsSink, enrichers ...LabelEnricher) *MetricsObserver {
	return &MetricsObserver{sink: sink, enrichers: enrichers}
}

// OnInvocationEvent records counts and latency per tenant/tool/outcome.
func (m *MetricsObserver) OnInvocationEvent(ctx context.Context, req protocol.InvocationRequest, evt protocol.InvocationEvent, invokeErr error, elapsed time.Duration) {
	if m == nil || m.sink == nil {
		return
	}
	outcome := classifyOutcome(evt, invokeErr)
	labels := map[string]string{
		"tenant":  req.TenantID,
		"tool":    req.Tool.Name,
		"outcome": outcome,
	}
	for _, enricher := range m.enrichers {
		for k, v := range enricher(ctx, req, evt, invokeErr) {
			labels[k] = v
		}
	}
	m.sink.IncCounter("mcp_invocations_total", labels)
	m.sink.ObserveHistogram("mcp_invocation_latency_seconds", elapsed.Seconds(), labels)
}

func classifyOutcome(evt protocol.InvocationEvent, invokeErr error) string {
	if invokeErr != nil {
		return "error"
	}
	if evt.Error != nil {
		if evt.Error.Code == mcperrors.ErrCancelled {
			return "cancelled"
		}
		return "error"
	}
	if evt.Type == protocol.EventResult {
		return "ok"
	}
	return "progress"
}
