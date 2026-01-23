package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/repository"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/service"
	"golang.org/x/time/rate"
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
	usagePublisher service.UsagePublisher
	policy         ExecutionPolicy
	rateLimiters   map[string]*rate.Limiter
	rateMu         sync.Mutex
}

// ExecutionPolicy controls outbound execution constraints.
type ExecutionPolicy struct {
	AllowedHosts      []string
	MaxPayloadBytes   int
	AllowedMethods    []string
	BlockedMethods    []string
	AllowedHeaders    []string
	BlockedHeaders    []string
	MaxQueryParams    int
	MaxTimeoutSeconds int
}

var (
	execBlockedTotal     *prometheus.CounterVec
	creditsConsumedTotal *prometheus.CounterVec
	rateLimitHitsTotal   *prometheus.CounterVec
	execMetricsOnce      sync.Once
)

func initExecutionMetrics() {
	execMetricsOnce.Do(func() {
		execBlockedTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tools_gateway_execution_blocked_total",
				Help: "Executions blocked before dispatch to external tools",
			},
			[]string{"reason", "tool", "tenant"},
		)

		creditsConsumedTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tools_gateway_credits_consumed_total",
				Help: "Credits consumed by tool executions",
			},
			[]string{"tenant", "tool"},
		)

		rateLimitHitsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tools_gateway_rate_limit_hits_total",
				Help: "Executions blocked by rate limiting",
			},
			[]string{"tenant", "tool"},
		)
		prometheus.MustRegister(execBlockedTotal)
		prometheus.MustRegister(creditsConsumedTotal)
		prometheus.MustRegister(rateLimitHitsTotal)
	})
}

// NewToolExecutorService creates a new ToolExecutorService
func NewToolExecutorService(
	toolRepo repository.ToolRepository,
	tenantToolRepo repository.TenantToolRepository,
	executionRepo repository.ToolExecutionRepository,
	validator service.SchemaValidator,
	httpClient service.HTTPClient,
	grpcClient service.GRPCClient,
	usagePublisher service.UsagePublisher,
	policy ExecutionPolicy,
) ToolExecutorService {
	if usagePublisher == nil {
		usagePublisher = service.NewNoopUsagePublisher()
	}
	return &toolExecutorServiceImpl{
		toolRepo:       toolRepo,
		tenantToolRepo: tenantToolRepo,
		executionRepo:  executionRepo,
		validator:      validator,
		httpClient:     httpClient,
		grpcClient:     grpcClient,
		usagePublisher: usagePublisher,
		policy:         policy,
		rateLimiters:   make(map[string]*rate.Limiter),
	}
}

// Execute executes a tool with the given input
func (s *toolExecutorServiceImpl) Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResponse, error) {
	startTime := time.Now()

	// Get tool
	tool, err := s.toolRepo.FindAccessibleByID(ctx, req.TenantID, req.ToolID)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}

	// Check if tool is active
	if !tool.IsActive {
		return nil, fmt.Errorf("tool '%s' is not active", tool.Name)
	}

	if err := s.enforceMethodPolicy(tool.Method, tool.Name, req.TenantID.String()); err != nil {
		return s.createErrorResponse(ctx, tool, req, entity.StatusError, err, 0)
	}

	if err := s.enforceHostAllowlist(tool, req.TenantID.String()); err != nil {
		return s.createErrorResponse(ctx, tool, req, entity.StatusError, err, 0)
	}

	if err := s.enforcePayloadLimit(req.Input, tool.Name, req.TenantID.String()); err != nil {
		return s.createErrorResponse(ctx, tool, req, entity.StatusError, err, 0)
	}

	if err := s.enforceHeaderPolicy(tool.Headers, tool.Name, req.TenantID.String()); err != nil {
		return s.createErrorResponse(ctx, tool, req, entity.StatusError, err, 0)
	}

	if err := s.enforceQueryParamLimit(req.Input, tool.Method, tool.Name, req.TenantID.String()); err != nil {
		return s.createErrorResponse(ctx, tool, req, entity.StatusError, err, 0)
	}

	// Get tenant-specific configuration (if exists)
	var authConfig json.RawMessage
	effectiveLimitPerMinute := tool.RateLimitPerMinute
	effectiveCreditCost := tool.CreditCost
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

		effectiveLimitPerMinute = tenantTool.GetRateLimitPerMinute(tool.RateLimitPerMinute)
		effectiveCreditCost = tenantTool.GetCreditCost(tool.CreditCost)
	}

	if err := s.enforceRateLimit(req.TenantID, tool.ID, effectiveLimitPerMinute, tool.Name); err != nil {
		return s.createErrorResponse(ctx, tool, req, entity.StatusRateLimited, err, 0)
	}

	// Validate input against schema
	if err := s.validator.ValidateInput(req.Input, tool.InputSchema); err != nil {
		return s.createErrorResponse(ctx, tool, req, entity.StatusError, err, 0)
	}

	// Cap timeout per tool to policy
	timeoutSeconds := tool.TimeoutSeconds
	if timeoutSeconds <= 0 || (s.policy.MaxTimeoutSeconds > 0 && timeoutSeconds > s.policy.MaxTimeoutSeconds) {
		timeoutSeconds = s.policy.MaxTimeoutSeconds
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	// Execute HTTP request
	httpResp, err := s.httpClient.Execute(ctxWithTimeout, tool, req.Input, authConfig)
	if err != nil {
		return s.createErrorResponse(ctx, tool, req, entity.StatusError, err, time.Since(startTime).Milliseconds())
	}

	// Validate output against schema
	if err := s.validator.ValidateOutput(httpResp.Body, tool.OutputSchema); err != nil {
		return s.createErrorResponse(ctx, tool, req, entity.StatusError,
			fmt.Errorf("output validation failed: %w", err), httpResp.LatencyMS)
	}

	// Create successful execution record
	return s.createSuccessResponse(ctx, tool, req, httpResp, effectiveCreditCost)
}

func (s *toolExecutorServiceImpl) enforceHostAllowlist(tool *entity.Tool, tenantID string) error {
	if len(s.policy.AllowedHosts) == 0 {
		return nil
	}

	parsed, err := url.Parse(tool.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid base_url: %w", err)
	}

	host := parsed.Hostname()
	for _, allowed := range s.policy.AllowedHosts {
		if strings.EqualFold(host, allowed) {
			return nil
		}
	}

	initExecutionMetrics()
	execBlockedTotal.WithLabelValues("host_not_allowed", tool.Name, tenantID).Inc()
	return fmt.Errorf("host '%s' not allowed", host)
}

func (s *toolExecutorServiceImpl) enforcePayloadLimit(input map[string]interface{}, toolName, tenantID string) error {
	if s.policy.MaxPayloadBytes <= 0 {
		return nil
	}

	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal input for size check: %w", err)
	}

	if len(body) > s.policy.MaxPayloadBytes {
		initExecutionMetrics()
		execBlockedTotal.WithLabelValues("payload_too_large", toolName, tenantID).Inc()
		return fmt.Errorf("payload too large: %d bytes (max %d)", len(body), s.policy.MaxPayloadBytes)
	}

	return nil
}

func (s *toolExecutorServiceImpl) enforceMethodPolicy(method, toolName, tenantID string) error {
	if len(s.policy.BlockedMethods) > 0 {
		for _, blocked := range s.policy.BlockedMethods {
			if strings.EqualFold(blocked, method) {
				initExecutionMetrics()
				execBlockedTotal.WithLabelValues("method_blocked", toolName, tenantID).Inc()
				return fmt.Errorf("method '%s' is blocked", method)
			}
		}
	}

	if len(s.policy.AllowedMethods) > 0 {
		for _, allowed := range s.policy.AllowedMethods {
			if strings.EqualFold(allowed, method) {
				return nil
			}
		}
		initExecutionMetrics()
		execBlockedTotal.WithLabelValues("method_not_allowed", toolName, tenantID).Inc()
		return fmt.Errorf("method '%s' is not in allowlist", method)
	}

	return nil
}

func (s *toolExecutorServiceImpl) enforceHeaderPolicy(headers json.RawMessage, toolName, tenantID string) error {
	if len(headers) == 0 {
		return nil
	}

	var headerMap map[string]string
	if err := json.Unmarshal(headers, &headerMap); err != nil {
		return fmt.Errorf("invalid headers configuration: %w", err)
	}

	for key := range headerMap {
		lowerKey := strings.ToLower(key)

		for _, blocked := range s.policy.BlockedHeaders {
			if strings.EqualFold(blocked, lowerKey) {
				initExecutionMetrics()
				execBlockedTotal.WithLabelValues("header_blocked", toolName, tenantID).Inc()
				return fmt.Errorf("header '%s' is blocked", key)
			}
		}

		if len(s.policy.AllowedHeaders) > 0 {
			allowed := false
			for _, allowedHeader := range s.policy.AllowedHeaders {
				if strings.EqualFold(allowedHeader, lowerKey) {
					allowed = true
					break
				}
			}
			if !allowed {
				initExecutionMetrics()
				execBlockedTotal.WithLabelValues("header_not_allowed", toolName, tenantID).Inc()
				return fmt.Errorf("header '%s' is not in allowlist", key)
			}
		}
	}

	return nil
}

func (s *toolExecutorServiceImpl) enforceQueryParamLimit(input map[string]interface{}, method, toolName, tenantID string) error {
	if !strings.EqualFold(method, entity.HTTPMethodGET) {
		return nil
	}

	if s.policy.MaxQueryParams <= 0 {
		return nil
	}

	if len(input) > s.policy.MaxQueryParams {
		initExecutionMetrics()
		execBlockedTotal.WithLabelValues("query_params_limit", toolName, tenantID).Inc()
		return fmt.Errorf("too many query params: %d (max %d)", len(input), s.policy.MaxQueryParams)
	}

	return nil
}

// createSuccessResponse creates a successful execution response and logs it
func (s *toolExecutorServiceImpl) createSuccessResponse(
	ctx context.Context,
	tool *entity.Tool,
	req *ExecutionRequest,
	httpResp *service.HTTPResponse,
	effectiveCreditCost int,
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
		CreditsConsumed: effectiveCreditCost,
		ExecutedAt:      time.Now(),
	}

	// Save execution
	if err := s.executionRepo.Create(ctx, execution); err != nil {
		// Log error but don't fail the execution
		fmt.Printf("failed to save execution: %v\n", err)
	}

	initExecutionMetrics()
	creditsConsumedTotal.WithLabelValues(req.TenantID.String(), tool.Name).Add(float64(effectiveCreditCost))

	if err := s.usagePublisher.PublishUsage(ctx, service.UsageEvent{
		TenantID: req.TenantID,
		ToolID:   tool.ID,
		UserID:   req.UserID,
		Credits:  effectiveCreditCost,
		Status:   entity.StatusSuccess,
		Latency:  httpResp.LatencyMS,
		At:       execution.ExecutedAt,
	}); err != nil {
		fmt.Printf("failed to publish usage: %v\n", err)
	}

	return &ExecutionResponse{
		ExecutionID:     execution.ID,
		ToolID:          tool.ID,
		ToolName:        tool.Name,
		Status:          entity.StatusSuccess,
		Output:          httpResp.Body,
		LatencyMS:       httpResp.LatencyMS,
		CreditsConsumed: effectiveCreditCost,
	}, nil
}

// enforceRateLimit applies a token bucket per tenant/tool to prevent abuse.
func (s *toolExecutorServiceImpl) enforceRateLimit(tenantID, toolID uuid.UUID, limitPerMinute int, toolName string) error {
	if limitPerMinute <= 0 {
		return nil
	}

	key := tenantID.String() + ":" + toolID.String()
	s.rateMu.Lock()
	limiter := s.rateLimiters[key]
	if limiter == nil {
		// Burst equals the per-minute allowance to tolerate short spikes.
		interval := time.Minute / time.Duration(limitPerMinute)
		limiter = rate.NewLimiter(rate.Every(interval), limitPerMinute)
		s.rateLimiters[key] = limiter
	}
	s.rateMu.Unlock()

	if limiter.Allow() {
		return nil
	}

	initExecutionMetrics()
	execBlockedTotal.WithLabelValues("rate_limit", toolName, tenantID.String()).Inc()
	rateLimitHitsTotal.WithLabelValues(tenantID.String(), toolName).Inc()
	return fmt.Errorf("rate limit exceeded: %d per minute", limitPerMinute)
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
