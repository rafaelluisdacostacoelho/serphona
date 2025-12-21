package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	var req dto.CreateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	tool := req.ToEntity()
	if err := h.toolService.CreateTool(c.Request.Context(), tool); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "creation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.FromEntity(tool))
}

// ListTools handles GET /api/v1/tools
func (h *ToolHandler) ListTools(c *gin.Context) {
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
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "list_failed",
			Message: err.Error(),
		})
		return
	}

	toolResponses := make([]*dto.ToolResponse, len(tools))
	for i, tool := range tools {
		toolResponses[i] = dto.FromEntity(tool)
	}

	c.JSON(http.StatusOK, dto.ListToolsResponse{
		Tools:  toolResponses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// GetTool handles GET /api/v1/tools/:id
func (h *ToolHandler) GetTool(c *gin.Context) {
	idParam := c.Param("id")
	toolID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tool ID format",
		})
		return
	}

	tool, err := h.toolService.GetTool(c.Request.Context(), toolID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "not_found",
			Message: "Tool not found",
		})
		return
	}

	c.JSON(http.StatusOK, dto.FromEntity(tool))
}

// ExecuteTool handles POST /api/v1/tools/:id/execute
func (h *ToolHandler) ExecuteTool(c *gin.Context) {
	idParam := c.Param("id")
	toolID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tool ID format",
		})
		return
	}

	var req dto.ExecuteToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Extract tenant_id and user_id from context (set by auth middleware)
	tenantID, _ := c.Get("tenant_id")
	userID, _ := c.Get("user_id")

	execReq := &usecase.ExecutionRequest{
		ToolID:   toolID,
		TenantID: tenantID.(uuid.UUID),
		UserID:   userID.(uuid.UUID),
		Input:    req.Input,
	}

	resp, err := h.executorService.Execute(c.Request.Context(), execReq)
	if err != nil {
		// Execution errors are also returned in response with error status
		if resp != nil {
			c.JSON(http.StatusOK, dto.ExecuteToolResponse{
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

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "execution_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.ExecuteToolResponse{
		ExecutionID:     resp.ExecutionID,
		ToolID:          resp.ToolID,
		ToolName:        resp.ToolName,
		Status:          resp.Status,
		Output:          resp.Output,
		LatencyMS:       resp.LatencyMS,
		CreditsConsumed: resp.CreditsConsumed,
	})
}
