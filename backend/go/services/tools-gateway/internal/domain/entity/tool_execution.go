package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ToolExecution represents a single execution of a tool
type ToolExecution struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserID   uuid.UUID `json:"user_id" gorm:"type:uuid;index"`
	ToolID   uuid.UUID `json:"tool_id" gorm:"type:uuid;not null;index"`

	// Request/Response
	InputData  json.RawMessage `json:"input_data" gorm:"type:jsonb;not null"`
	OutputData json.RawMessage `json:"output_data,omitempty" gorm:"type:jsonb"`

	// Execution Details
	Status       string `json:"status" gorm:"not null;index"` // success, error, timeout, rate_limited
	ErrorMessage string `json:"error_message,omitempty"`
	LatencyMS    int    `json:"latency_ms"`

	// Billing
	CreditsConsumed int `json:"credits_consumed" gorm:"default:0"`

	// Metadata
	ExecutedAt time.Time `json:"executed_at" gorm:"default:now();index"`
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`

	// Relations
	Tool *Tool `json:"tool,omitempty" gorm:"foreignKey:ToolID"`
}

// TableName specifies the table name for GORM
func (ToolExecution) TableName() string {
	return "tool_executions"
}

// ExecutionStatus constants
const (
	StatusSuccess     = "success"
	StatusError       = "error"
	StatusTimeout     = "timeout"
	StatusRateLimited = "rate_limited"
	StatusInProgress  = "in_progress"
)

// IsSuccessful returns true if execution was successful
func (te *ToolExecution) IsSuccessful() bool {
	return te.Status == StatusSuccess
}

// HasError returns true if execution failed
func (te *ToolExecution) HasError() bool {
	return te.Status == StatusError || te.Status == StatusTimeout
}
