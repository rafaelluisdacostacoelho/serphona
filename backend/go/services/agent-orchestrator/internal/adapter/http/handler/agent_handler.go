package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/adapter/http/dto"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/usecase"
)

// AgentHandler handles agent-related HTTP requests
type AgentHandler struct {
	agentService usecase.AgentService
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(agentService usecase.AgentService) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
	}
}

// CreateAgent handles POST /api/v1/agents
func (h *AgentHandler) CreateAgent(c *gin.Context) {
	ctx := c.Request.Context()

	var req dto.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", nil)
		return
	}

	// Convert DTO to entity
	agent, err := req.ToAgentEntity()
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", "invalid agent payload", map[string]string{"reason": err.Error()})
		return
	}

	// Create agent
	if err := h.agentService.CreateAgent(c.Request.Context(), agent); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create agent", map[string]string{"reason": err.Error()})
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusCreated, dto.ToAgentResponse(agent))
}

// GetAgent handles GET /api/v1/agents/:id
func (h *AgentHandler) GetAgent(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid agent ID", nil)
		return
	}

	agent, err := h.agentService.GetAgent(c.Request.Context(), id)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.ToAgentResponse(agent))
}

// UpdateAgent handles PUT /api/v1/agents/:id
func (h *AgentHandler) UpdateAgent(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid agent ID", nil)
		return
	}

	var req dto.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", nil)
		return
	}

	// Get existing agent
	agent, err := h.agentService.GetAgent(c.Request.Context(), id)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}

	// Update fields
	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.DisplayName != "" {
		agent.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		agent.Description = req.Description
	}
	if req.SystemPrompt != "" {
		agent.SystemPrompt = req.SystemPrompt
	}
	if req.Model != "" {
		agent.Model = req.Model
	}
	if req.Temperature != nil {
		agent.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		agent.MaxTokens = *req.MaxTokens
	}
	if req.Tools != nil {
		agent.Tools = req.Tools
	}
	if req.IsActive != nil {
		agent.IsActive = *req.IsActive
	}

	// Update agent
	if err := h.agentService.UpdateAgent(c.Request.Context(), agent); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update agent", map[string]string{"reason": err.Error()})
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.ToAgentResponse(agent))
}

// DeleteAgent handles DELETE /api/v1/agents/:id
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid agent ID", nil)
		return
	}

	if err := h.agentService.DeleteAgent(c.Request.Context(), id); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete agent", map[string]string{"reason": err.Error()})
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, statusMessageResponse{Message: "agent deleted successfully"})
}

// ListAgents handles GET /api/v1/agents
func (h *AgentHandler) ListAgents(c *gin.Context) {
	ctx := c.Request.Context()
	// Get query parameters
	tenantIDStr := c.Query("tenant_id")
	if tenantIDStr == "" {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", "tenant_id is required", nil)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_TENANT_ID", "invalid tenant ID", nil)
		return
	}

	activeOnly := c.Query("active") == "true"

	agents, err := h.agentService.ListAgents(c.Request.Context(), tenantID, activeOnly)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list agents", map[string]string{"reason": err.Error()})
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, listAgentsResponse{
		Agents: dto.ToAgentResponseList(agents),
		Count:  len(agents),
	})
}

// ActivateAgent handles POST /api/v1/agents/:id/activate
func (h *AgentHandler) ActivateAgent(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid agent ID", nil)
		return
	}

	if err := h.agentService.ActivateAgent(c.Request.Context(), id); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to activate agent", map[string]string{"reason": err.Error()})
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, statusMessageResponse{Message: "agent activated successfully"})
}

// DeactivateAgent handles POST /api/v1/agents/:id/deactivate
func (h *AgentHandler) DeactivateAgent(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid agent ID", nil)
		return
	}

	if err := h.agentService.DeactivateAgent(c.Request.Context(), id); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to deactivate agent", map[string]string{"reason": err.Error()})
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, statusMessageResponse{Message: "agent deactivated successfully"})
}

type listAgentsResponse struct {
	Agents []*dto.AgentResponse `json:"agents"`
	Count  int                  `json:"count"`
}

type statusMessageResponse struct {
	Message string `json:"message"`
}
