package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

// ToolExecutionRepository defines the interface for tool execution data operations
type ToolExecutionRepository interface {
	// Create creates a new tool execution record
	Create(ctx context.Context, execution *entity.ToolExecution) error

	// FindByID finds a tool execution by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entity.ToolExecution, error)

	// FindByTenant finds all executions for a tenant with filters
	FindByTenant(ctx context.Context, tenantID uuid.UUID, filters ExecutionFilters) ([]*entity.ToolExecution, int64, error)

	// FindByTool finds all executions for a specific tool
	FindByTool(ctx context.Context, toolID uuid.UUID, filters ExecutionFilters) ([]*entity.ToolExecution, int64, error)

	// Update updates an existing execution (useful for updating status/output)
	Update(ctx context.Context, execution *entity.ToolExecution) error

	// GetStats gets execution statistics
	GetStats(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*ExecutionStats, error)

	// GetCostBreakdown gets cost breakdown by tool
	GetCostBreakdown(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]*CostBreakdown, error)
}

// ExecutionFilters represents filters for execution queries
type ExecutionFilters struct {
	ToolID   *uuid.UUID
	UserID   *uuid.UUID
	Status   string
	FromDate *time.Time
	ToDate   *time.Time
	Limit    int
	Offset   int
}

// ExecutionStats represents execution statistics
type ExecutionStats struct {
	TotalExecutions  int64   `json:"total_executions"`
	SuccessfulCount  int64   `json:"successful_count"`
	FailedCount      int64   `json:"failed_count"`
	TimeoutCount     int64   `json:"timeout_count"`
	RateLimitedCount int64   `json:"rate_limited_count"`
	TotalCredits     int64   `json:"total_credits"`
	AverageLatencyMS float64 `json:"average_latency_ms"`
	SuccessRate      float64 `json:"success_rate"`
}

// CostBreakdown represents cost breakdown by tool
type CostBreakdown struct {
	ToolID         uuid.UUID `json:"tool_id"`
	ToolName       string    `json:"tool_name"`
	ExecutionCount int64     `json:"execution_count"`
	TotalCredits   int64     `json:"total_credits"`
	SuccessCount   int64     `json:"success_count"`
	FailedCount    int64     `json:"failed_count"`
	AverageLatency float64   `json:"average_latency"`
}
