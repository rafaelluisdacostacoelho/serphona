package events

import "time"

// ToolRegisteredEvent captures registration of a tool integration for a tenant.
type ToolRegisteredEvent struct {
	ToolID       string            `json:"tool_id"`
	TenantID     string            `json:"tenant_id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	RegisteredAt time.Time         `json:"registered_at"`
	RegisteredBy string            `json:"registered_by,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ToolInvokedEvent captures when a tool is invoked, including correlation.
type ToolInvokedEvent struct {
	ToolID        string                 `json:"tool_id"`
	TenantID      string                 `json:"tenant_id"`
	Action        string                 `json:"action"`
	InvokedAt     time.Time              `json:"invoked_at"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
}

// ToolCompletedEvent captures successful completion of a tool action.
type ToolCompletedEvent struct {
	ToolID        string                 `json:"tool_id"`
	TenantID      string                 `json:"tenant_id"`
	Action        string                 `json:"action"`
	Result        map[string]interface{} `json:"result,omitempty"`
	DurationMs    int64                  `json:"duration_ms,omitempty"`
	CompletedAt   time.Time              `json:"completed_at"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
}

// ToolFailedEvent documents a tool invocation failure with diagnostics.
type ToolFailedEvent struct {
	ToolID        string    `json:"tool_id"`
	TenantID      string    `json:"tenant_id"`
	Action        string    `json:"action"`
	Error         string    `json:"error"`
	DurationMs    int64     `json:"duration_ms,omitempty"`
	FailedAt      time.Time `json:"failed_at"`
	Context       string    `json:"context,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
}
