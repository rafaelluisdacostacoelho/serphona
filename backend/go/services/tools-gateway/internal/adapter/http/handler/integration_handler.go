// Package handler contains HTTP handlers.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	var req dto.CreateIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Get tenant ID from context (set by auth middleware)
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "unauthorized",
			Message: "tenant_id not found in context",
		})
		return
	}

	// Convert DTO to entity
	integration := req.ToEntity(tenantID.(uuid.UUID))

	// Create integration
	if err := h.integrationService.CreateIntegration(c.Request.Context(), integration); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "creation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.IntegrationResponse{Integration: integration})
}

// GetIntegration handles GET /api/v1/integrations/:id.
func (h *IntegrationHandler) GetIntegration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "invalid integration ID",
		})
		return
	}

	integration, err := h.integrationService.GetIntegration(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "not_found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.IntegrationResponse{Integration: integration})
}

// ListIntegrations handles GET /api/v1/integrations.
func (h *IntegrationHandler) ListIntegrations(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	var filters repository.IntegrationFilters
	tid := tenantID.(uuid.UUID)
	filters.TenantID = &tid

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
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "list_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.ListIntegrationsResponse{
		Integrations: integrations,
		Total:        total,
		Limit:        filters.Limit,
		Offset:       filters.Offset,
	})
}

// UpdateIntegration handles PUT /api/v1/integrations/:id.
func (h *IntegrationHandler) UpdateIntegration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "invalid integration ID",
		})
		return
	}

	var req dto.UpdateIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	tenantID, _ := c.Get("tenant_id")

	integration := req.ToEntity(id, tenantID.(uuid.UUID))

	if err := h.integrationService.UpdateIntegration(c.Request.Context(), integration); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "update_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.IntegrationResponse{Integration: integration})
}

// DeleteIntegration handles DELETE /api/v1/integrations/:id.
func (h *IntegrationHandler) DeleteIntegration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "invalid integration ID",
		})
		return
	}

	if err := h.integrationService.DeleteIntegration(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "deletion_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ActivateIntegration handles POST /api/v1/integrations/:id/activate.
func (h *IntegrationHandler) ActivateIntegration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "invalid integration ID",
		})
		return
	}

	if err := h.integrationService.ActivateIntegration(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "activation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "integration activated"})
}

// DeactivateIntegration handles POST /api/v1/integrations/:id/deactivate.
func (h *IntegrationHandler) DeactivateIntegration(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "invalid integration ID",
		})
		return
	}

	if err := h.integrationService.DeactivateIntegration(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "deactivation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "integration deactivated"})
}
