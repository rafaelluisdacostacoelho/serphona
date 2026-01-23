// Package handler contains HTTP handlers.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/adapter/http/dto"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/repository"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/usecase"
)

// IntegrationHandler handles integration HTTP requests.
type IntegrationHandler struct {
	integrationService usecase.IntegrationService
}

// NewIntegrationHandler creates a new IntegrationHandler.
func NewIntegrationHandler(integrationService usecase.IntegrationService) *IntegrationHandler {
	return &IntegrationHandler{
		integrationService: integrationService,
	}
}

// CreateIntegration handles POST /api/v1/integrations.
func (h *IntegrationHandler) CreateIntegration(c *gin.Context) {
	ctx := c.Request.Context()
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "claims not found in context", nil)
		return
	}

	var req dto.CreateIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", nil)
		return
	}

	tenantUUID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_TENANT_ID", "tenant_id has invalid format", nil)
		return
	}

	// Convert DTO to entity
	integration := req.ToEntity(tenantUUID)

	// Create integration
	if err := h.integrationService.CreateIntegration(c.Request.Context(), integration); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "CREATION_FAILED", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusCreated, dto.IntegrationResponse{Integration: integration})
}

// GetIntegration handles GET /api/v1/integrations/:id.
func (h *IntegrationHandler) GetIntegration(c *gin.Context) {
	ctx := c.Request.Context()
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "claims not found in context", nil)
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_TENANT_ID", "tenant_id has invalid format", nil)
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid integration ID", nil)
		return
	}

	integration, err := h.integrationService.GetIntegration(c.Request.Context(), tenantID, id)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.IntegrationResponse{Integration: integration})
}

// ListIntegrations handles GET /api/v1/integrations.
func (h *IntegrationHandler) ListIntegrations(c *gin.Context) {
	ctx := c.Request.Context()
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "claims not found in context", nil)
		return
	}

	tenantUUID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_TENANT_ID", "tenant_id has invalid format", nil)
		return
	}

	var filters repository.IntegrationFilters
	filters.TenantID = &tenantUUID

	if provider := c.Query("provider"); provider != "" {
		filters.Provider = provider
	}

	if intType := c.Query("type"); intType != "" {
		filters.Type = entity.IntegrationType(intType)
	}

	if c.Query("is_active") != "" {
		isActive := c.Query("is_active") == "true"
		filters.IsActive = &isActive
	}

	filters.Limit = 20
	if c.Query("limit") != "" {
		c.ShouldBindQuery(&filters)
	}

	integrations, total, err := h.integrationService.ListIntegrations(c.Request.Context(), filters)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "LIST_FAILED", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.ListIntegrationsResponse{
		Integrations: integrations,
		Total:        total,
		Limit:        filters.Limit,
		Offset:       filters.Offset,
	}, response.WithPagination(buildPagination(total, filters.Limit, filters.Offset)))
}

// UpdateIntegration handles PUT /api/v1/integrations/:id.
func (h *IntegrationHandler) UpdateIntegration(c *gin.Context) {
	ctx := c.Request.Context()
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "claims not found in context", nil)
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_TENANT_ID", "tenant_id has invalid format", nil)
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid integration ID", nil)
		return
	}

	var req dto.UpdateIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body", nil)
		return
	}

	integration := req.ToEntity(id, tenantID)

	if err := h.integrationService.UpdateIntegration(c.Request.Context(), tenantID, integration); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "UPDATE_FAILED", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.IntegrationResponse{Integration: integration})
}

// DeleteIntegration handles DELETE /api/v1/integrations/:id.
func (h *IntegrationHandler) DeleteIntegration(c *gin.Context) {
	ctx := c.Request.Context()
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "claims not found in context", nil)
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_TENANT_ID", "tenant_id has invalid format", nil)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid integration ID", nil)
		return
	}

	if err := h.integrationService.DeleteIntegration(c.Request.Context(), tenantID, id); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "DELETION_FAILED", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusNoContent, nil)
}

// ActivateIntegration handles POST /api/v1/integrations/:id/activate.
func (h *IntegrationHandler) ActivateIntegration(c *gin.Context) {
	ctx := c.Request.Context()
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "claims not found in context", nil)
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_TENANT_ID", "tenant_id has invalid format", nil)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid integration ID", nil)
		return
	}

	if err := h.integrationService.ActivateIntegration(c.Request.Context(), tenantID, id); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "ACTIVATION_FAILED", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, statusMessageResponse{Message: "integration activated"})
}

// DeactivateIntegration handles POST /api/v1/integrations/:id/deactivate.
func (h *IntegrationHandler) DeactivateIntegration(c *gin.Context) {
	ctx := c.Request.Context()
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "claims not found in context", nil)
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_TENANT_ID", "tenant_id has invalid format", nil)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid integration ID", nil)
		return
	}

	if err := h.integrationService.DeactivateIntegration(c.Request.Context(), tenantID, id); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "DEACTIVATION_FAILED", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, statusMessageResponse{Message: "integration deactivated"})
}
