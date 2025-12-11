package service

import (
	"context"

	"github.com/google/uuid"
)

// ToolExecutionRequest represents a request to execute a tool
type ToolExecutionRequest struct {
	ToolName   string                 `json:"tool_name"`
	Parameters map[string]interface{} `json:"parameters"`
	TenantID   uuid.UUID              `json:"tenant_id"`
	UserID     string                 `json:"user_id"`
}

// ToolExecutionResponse represents the response from tool execution
type ToolExecutionResponse struct {
	ExecutionID string                 `json:"execution_id"`
	ToolName    string                 `json:"tool_name"`
	Status      string                 `json:"status"` // success, error, timeout
	Result      map[string]interface{} `json:"result"`
	Error       string                 `json:"error,omitempty"`
	Duration    int64                  `json:"duration_ms"`
}

// ToolsClient defines the interface for interacting with the Tools Gateway service
type ToolsClient interface {
	// ExecuteTool executes a tool with the given parameters
	ExecuteTool(ctx context.Context, request *ToolExecutionRequest) (*ToolExecutionResponse, error)

	// GetAvailableTools retrieves the list of available tools for a tenant
	GetAvailableTools(ctx context.Context, tenantID uuid.UUID) ([]string, error)

	// ValidateTool checks if a tool is available and valid
	ValidateTool(ctx context.Context, tenantID uuid.UUID, toolName string) (bool, error)
}
