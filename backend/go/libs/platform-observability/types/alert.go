package types

import "time"

// AlertEvent representa um alerta de anomalia ou ML.
type AlertEvent struct {
    AlertID       string         `json:"alert_id"`
    TenantID      string         `json:"tenant_id"`
    EventType     string         `json:"event_type"` // anomaly.detected, alert.ml
    Source        string         `json:"source"`
    Severity      string         `json:"severity"`
    Message       string         `json:"message"`
    Score         float64        `json:"score,omitempty"`
    Current       int            `json:"current,omitempty"`
    Mean          float64        `json:"mean,omitempty"`
    Std           float64        `json:"std,omitempty"`
    WindowMinutes int            `json:"window_minutes,omitempty"`
    Key           string         `json:"key,omitempty"`
    Metadata      map[string]any `json:"metadata,omitempty"`
    Timestamp     time.Time      `json:"timestamp"`
}
