package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/repository"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/service"
)

// ExecutionRequest represents a tool execution request
type ExecutionRequest struct {
	ToolID   uuid.UUID              `json:"tool_id"`
	TenantID uuid.UUID              `json:"tenant_id"`
	UserID   uuid.UUID              `json:"user_id"`
	Input    map[string]interface{} `json:"input"`
}

// ExecutionResponse represents a tool execution response
type ExecutionResponse struct {
	ExecutionID     uuid.UUID              `json:"execution_id"`
	ToolID          uuid.UUID              `json:"tool_id"`
	ToolName        string                 `json:"tool_name"`
	Status          string                 `json:"status"`
	Output          map[string]interface{} `json:"output,omitempty"`
	Error           string                 `json:"error,omitempty"`
	LatencyMS       int64                  `json:"latency_ms"`
	CreditsConsumed int                    `json:"credits_consumed"`
}

// ToolExecutorService defines the interface for tool execution
type ToolExecutorService interface {
	// Execute executes a tool with the given input
	Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResponse, error)

	// GetExecution gets an execution by ID
	GetExecution(ctx context.Context, executionID uuid.UUID) (*entity.ToolExecution, error)

	// ListExecutions lists executions with filters
	ListExecutions(ctx context.Context, tenantID uuid.UUID, filters repository.ExecutionFilters) ([]*entity.ToolExecution, int64, error)

	// GetExecutionStats gets execution statistics for a tenant
	GetExecutionStats(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*repository.ExecutionStats, error)

	// GetCostBreakdown gets cost breakdown by tool for a tenant
	GetCostBreakdown(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]*repository.CostBreakdown, error)
}

// toolExecutorServiceImpl implements ToolExecutorService
type toolExecutorServiceImpl struct {
	toolRepo       repository.ToolRepository
	tenantToolRepo repository.TenantToolRepository
	executionRepo  repository.ToolExecutionRepository
	validator      service.SchemaValidator
	httpClient     service.HTTPClient
	grpcClient     service.GRPCClient
}

// NewToolExecutorService creates a new ToolExecutorService
func NewToolExecutorService(
	toolRepo repository.ToolRepository,
	tenantToolRepo repository.TenantToolRepository,
	executionRepo repository.ToolExecutionRepository,
	validator service.SchemaValidator,
	httpClient service.HTTPClient,
	grpcClient service.GRPCClient,
) ToolExecutorService {
	return &toolExecutorServiceImpl{
		toolRepo:       toolRepo,
		tenantToolRepo: tenantToolRepo,
		executionRepo:  executionRepo,
		validator:      validator,
		httpClient:     httpClient,
		grpcClient:     grpcClient,
	}
}

// Execute executes a tool with the given input
func (s *toolExecutorServiceImpl) Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResponse, error) {
	startTime := time.Now()

	// Get tool
	tool, err := s.toolRepo.FindByID(ctx, req.ToolID)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}

	// Check if tool is active
	if !tool.IsActive {
		return nil, fmt.Errorf("tool '%s' is not active", tool.Name)
	}

	// Get tenant-specific configuration (if exists)
	var authConfig json.RawMessage
	tenantTool, err := s.tenantToolRepo.FindByTenantAndTool(ctx, req.TenantID, req.ToolID)
	if err == nil && tenantTool != nil {
		// Check if tool is enabled for tenant
		if !tenantTool.IsEnabled {
			return nil, fmt.Errorf("tool '%s' is not enabled for this tenant", tool.Name)
		}

		// Check if user is allowed
		if !tenantTool.IsUserAllowed(req.UserID) {
			return nil, fmt.Errorf("user is not allowed to use tool '%s'", tool.Name)
		}

		// Use tenant-specific auth config if available
		if len(tenantTool.CustomAuthConfig) > 0 {
			authConfig = tenantTool.CustomAuthConfig
		}
	}

	// Validate input against schema
	if err := s.validator.ValidateInput(req.Input, tool.InputSchema); err != nil {
		return s.createErrorResponse(ctx, tool, req, entity.StatusError, err, 0)
	}

	// Execute HTTP request
	httpResp, err := s.httpClient.Execute(ctx, tool, req.Input, authConfig)
	if err != nil {
		return s.createErrorResponse(ctx, tool, req, entity.StatusError, err, time.Since(startTime).Milliseconds())
	}

	// Validate output against schema
	if err := s.validator.ValidateOutput(httpResp.Body, tool.OutputSchema); err != nil {
		return s.createErrorResponse(ctx, tool, req, entity.StatusError,
			fmt.Errorf("output validation failed: %w", err), httpResp.LatencyMS)
	}

	// Create successful execution record
	return s.createSuccessResponse(ctx, tool, req, httpResp)
}

// createSuccessResponse creates a successful execution response and logs it
func (s *toolExecutorServiceImpl) createSuccessResponse(
	ctx context.Context,
	tool *entity.Tool,
	req *ExecutionRequest,
	httpResp *service.HTTPResponse,
) (*ExecutionResponse, error) {
	// Marshal input and output
	inputJSON, _ := json.Marshal(req.Input)
	outputJSON, _ := json.Marshal(httpResp.Body)

	// Create execution record
	execution := &entity.ToolExecution{
		ID:              uuid.New(),
		TenantID:        req.TenantID,
		UserID:          req.UserID,
		ToolID:          tool.ID,
		InputData:       inputJSON,
		OutputData:      outputJSON,
		Status:          entity.StatusSuccess,
		LatencyMS:       int(httpResp.LatencyMS),
		CreditsConsumed: tool.CreditCost,
		ExecutedAt:      time.Now(),
	}

	// Save execution
	if err := s.executionRepo.Create(ctx, execution); err != nil {
		// Log error but don't fail the execution
		fmt.Printf("failed to save execution: %v\n", err)
	}

	return &ExecutionResponse{
		ExecutionID:     execution.ID,
		ToolID:          tool.ID,
		ToolName:        tool.Name,
		Status:          entity.StatusSuccess,
		Output:          httpResp.Body,
		LatencyMS:       httpResp.LatencyMS,
		CreditsConsumed: tool.CreditCost,
	}, nil
}

// createErrorResponse creates an error response and logs it
func (s *toolExecutorServiceImpl) createErrorResponse(
	ctx context.Context,
	tool *entity.Tool,
	req *ExecutionRequest,
	status string,
	err error,
	latencyMS int64,
) (*ExecutionResponse, error) {
	// Marshal input
	inputJSON, _ := json.Marshal(req.Input)

	// Create execution record
	execution := &entity.ToolExecution{
		ID:              uuid.New(),
		TenantID:        req.TenantID,
		UserID:          req.UserID,
		ToolID:          tool.ID,
		InputData:       inputJSON,
		Status:          status,
		ErrorMessage:    err.Error(),
		LatencyMS:       int(latencyMS),
		CreditsConsumed: 0, // Don't charge for failed executions
		ExecutedAt:      time.Now(),
	}

	// Save execution
	if saveErr := s.executionRepo.Create(ctx, execution); saveErr != nil {
		fmt.Printf("failed to save error execution: %v\n", saveErr)
	}

	return &ExecutionResponse{
		ExecutionID:     execution.ID,
		ToolID:          tool.ID,
		ToolName:        tool.Name,
		Status:          status,
		Error:           err.Error(),
		LatencyMS:       latencyMS,
		CreditsConsumed: 0,
	}, err
}

// GetExecution gets an execution by ID
func (s *toolExecutorServiceImpl) GetExecution(ctx context.Context, executionID uuid.UUID) (*entity.ToolExecution, error) {
	return s.executionRepo.FindByID(ctx, executionID)
}

// ListExecutions lists executions with filters
func (s *toolExecutorServiceImpl) ListExecutions(
	ctx context.Context,
	tenantID uuid.UUID,
	filters repository.ExecutionFilters,
) ([]*entity.ToolExecution, int64, error) {
	return s.executionRepo.FindByTenant(ctx, tenantID, filters)
}

// GetExecutionStats gets execution statistics for a tenant
func (s *toolExecutorServiceImpl) GetExecutionStats(
	ctx context.Context,
	tenantID uuid.UUID,
	from, to time.Time,
) (*repository.ExecutionStats, error) {
	return s.executionRepo.GetStats(ctx, tenantID, from, to)
}

// GetCostBreakdown gets cost breakdown by tool for a tenant
func (s *toolExecutorServiceImpl) GetCostBreakdown(
	ctx context.Context,
	tenantID uuid.UUID,
	from, to time.Time,
) ([]*repository.CostBreakdown, error) {
	return s.executionRepo.GetCostBreakdown(ctx, tenantID, from, to)
}
