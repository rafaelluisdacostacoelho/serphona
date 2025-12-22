package metrics

import (
    "net/http"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    conversationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "obs_conversation_events_total",
        Help: "Count of conversation lifecycle events",
    }, []string{"event", "tenant"})

    interactionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "obs_interactions_total",
        Help: "Count of interactions by speaker",
    }, []string{"speaker", "tenant"})

    decisionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "obs_decisions_total",
        Help: "Count of decisions taken",
    }, []string{"type", "tenant"})
)

// TrackConversation increments conversation event counters.
func TrackConversation(event, tenant string) {
    conversationsTotal.WithLabelValues(event, tenant).Inc()
}

// TrackInteraction increments interaction counters.
func TrackInteraction(speaker, tenant string) {
    interactionsTotal.WithLabelValues(speaker, tenant).Inc()
}

// TrackDecision increments decision counters.
func TrackDecision(kind, tenant string) {
    decisionsTotal.WithLabelValues(kind, tenant).Inc()
}

// Handler exposes the Prometheus HTTP handler.
func Handler() http.Handler {
	return promhttp.Handler()
}
