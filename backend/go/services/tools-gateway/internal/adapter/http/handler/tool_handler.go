package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/adapter/http/dto"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/repository"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/usecase"
)

// ToolHandler handles tool-related HTTP requests
type ToolHandler struct {
	toolService     usecase.ToolService
	executorService usecase.ToolExecutorService
}

// NewToolHandler creates a new ToolHandler
func NewToolHandler(
	toolService usecase.ToolService,
	executorService usecase.ToolExecutorService,
) *ToolHandler {
	return &ToolHandler{
		toolService:     toolService,
		executorService: executorService,
	}
}

// CreateTool handles POST /api/v1/tools
func (h *ToolHandler) CreateTool(c *gin.Context) {
	ctx := c.Request.Context()
	var req dto.CreateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", nil)
		return
	}

	tool := req.ToEntity()
	if err := h.toolService.CreateTool(c.Request.Context(), tool); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "CREATION_FAILED", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusCreated, dto.FromEntity(tool))
}

// ListTools handles GET /api/v1/tools
func (h *ToolHandler) ListTools(c *gin.Context) {
	ctx := c.Request.Context()
	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	category := c.Query("category")
	search := c.Query("search")

	var isActive *bool
	if c.Query("is_active") != "" {
		active := c.Query("is_active") == "true"
		isActive = &active
	}

	filters := repository.ToolFilters{
		Category: category,
		IsActive: isActive,
		Search:   search,
		Limit:    limit,
		Offset:   offset,
	}

	tools, total, err := h.toolService.ListTools(c.Request.Context(), filters)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "LIST_FAILED", err.Error(), nil)
		return
	}

	toolResponses := make([]*dto.ToolResponse, len(tools))
	for i, tool := range tools {
		toolResponses[i] = dto.FromEntity(tool)
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.ListToolsResponse{
		Tools:  toolResponses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, response.WithPagination(buildPagination(total, limit, offset)))
}

// GetTool handles GET /api/v1/tools/:id
func (h *ToolHandler) GetTool(c *gin.Context) {
	ctx := c.Request.Context()
	idParam := c.Param("id")
	toolID, err := uuid.Parse(idParam)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid tool ID format", nil)
		return
	}

	tool, err := h.toolService.GetTool(c.Request.Context(), toolID)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusNotFound, "NOT_FOUND", "tool not found", nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.FromEntity(tool))
}

// ExecuteTool handles POST /api/v1/tools/:id/execute
func (h *ToolHandler) ExecuteTool(c *gin.Context) {
	ctx := c.Request.Context()
	idParam := c.Param("id")
	toolID, err := uuid.Parse(idParam)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid tool ID format", nil)
		return
	}

	var req dto.ExecuteToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", nil)
		return
	}

	tenantVal, ok := c.Get("tenant_id")
	if !ok {
		response.WriteError(ctx, c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "tenant_id not found in context", nil)
		return
	}
	tenantID, ok := tenantVal.(uuid.UUID)
	if !ok {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_TENANT_ID", "tenant_id has invalid format", nil)
		return
	}

	userVal, ok := c.Get("user_id")
	if !ok {
		response.WriteError(ctx, c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "user_id not found in context", nil)
		return
	}
	userID, ok := userVal.(uuid.UUID)
	if !ok {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_USER_ID", "user_id has invalid format", nil)
		return
	}

	execReq := &usecase.ExecutionRequest{
		ToolID:   toolID,
		TenantID: tenantID,
		UserID:   userID,
		Input:    req.Input,
	}

	// Set tenant context for outbound requests
	ctxWithTenant := middleware.WithTenantID(ctx, tenantID.String())

	resp, err := h.executorService.Execute(ctxWithTenant, execReq)
	if err != nil {
		// Execution errors are also returned in response with error status
		if resp != nil {
			response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.ExecuteToolResponse{
				ExecutionID:     resp.ExecutionID,
				ToolID:          resp.ToolID,
				ToolName:        resp.ToolName,
				Status:          resp.Status,
				Error:           resp.Error,
				LatencyMS:       resp.LatencyMS,
				CreditsConsumed: resp.CreditsConsumed,
			})
			return
		}

		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "EXECUTION_FAILED", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.ExecuteToolResponse{
		ExecutionID:     resp.ExecutionID,
		ToolID:          resp.ToolID,
		ToolName:        resp.ToolName,
		Status:          resp.Status,
		Output:          resp.Output,
		LatencyMS:       resp.LatencyMS,
		CreditsConsumed: resp.CreditsConsumed,
	})
}
