package dto

import (
	"github.com/google/uuid"
)

// ExecuteToolRequest represents the request to execute a tool
type ExecuteToolRequest struct {
	Input map[string]interface{} `json:"input" binding:"required"`
}

// ExecuteToolResponse represents the response from tool execution
type ExecuteToolResponse struct {
	ExecutionID     uuid.UUID              `json:"execution_id"`
	ToolID          uuid.UUID              `json:"tool_id"`
	ToolName        string                 `json:"tool_name"`
	Status          string                 `json:"status"`
	Output          map[string]interface{} `json:"output,omitempty"`
	Error           string                 `json:"error,omitempty"`
	LatencyMS       int64                  `json:"latency_ms"`
	CreditsConsumed int                    `json:"credits_consumed"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}
