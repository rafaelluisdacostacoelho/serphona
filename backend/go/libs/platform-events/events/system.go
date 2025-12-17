package events

import "time"

// SystemHealthCheckEvent reports liveness/readiness results for a service.
type SystemHealthCheckEvent struct {
	Service   string            `json:"service"`
	Status    string            `json:"status"`
	CheckedAt time.Time         `json:"checked_at"`
	Details   map[string]string `json:"details,omitempty"`
}

// SystemAlertEvent signals an alert raised by a service or subsystem.
type SystemAlertEvent struct {
	AlertID   string            `json:"alert_id"`
	Severity  string            `json:"severity"`
	Service   string            `json:"service"`
	Message   string            `json:"message"`
	CreatedAt time.Time         `json:"created_at"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ConfigurationUpdatedEvent tracks configuration changes applied to a service.
type ConfigurationUpdatedEvent struct {
	Service   string            `json:"service"`
	UpdatedBy string            `json:"updated_by,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
	Changes   map[string]string `json:"changes,omitempty"`
}
