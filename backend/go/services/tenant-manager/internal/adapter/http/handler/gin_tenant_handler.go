// Package handler contains HTTP request handlers.
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"tenant-manager/internal/application/tenant"
	apperrors "tenant-manager/pkg/errors"
	"go.uber.org/zap"
)

// GinTenantHandler wraps TenantHandler for Gin framework.
type GinTenantHandler struct {
	service   *tenant.Service
	logger    *zap.Logger
	validator *validator.Validate
}

// NewGinTenantHandler creates a new Gin-compatible tenant handler.
func NewGinTenantHandler(service *tenant.Service, logger *zap.Logger) *GinTenantHandler {
	return &GinTenantHandler{
		service:   service,
		logger:    logger,
		validator: validator.New(),
	}
}

// Create handles POST /api/v1/tenants.
func (h *GinTenantHandler) Create(c *gin.Context) {
	var req CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_request", "Invalid JSON body", nil)
		return
	}
	if err := h.validator.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors[err.Field()] = getValidationMessage(err)
		}
		h.respondError(c, http.StatusBadRequest, "validation_error", "Validation failed", validationErrors)
		return
	}

	cmd := tenant.CreateTenantCommand{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Plan:         req.Plan,
		BillingEmail: req.BillingEmail,
		Industry:     req.Metadata.Industry,
		CompanySize:  req.Metadata.CompanySize,
		Website:      req.Metadata.Website,
	}
	result, err := h.service.CreateTenant(c.Request.Context(), cmd)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	h.logger.Info("tenant created", zap.String("tenant_id", result.ID.String()))
	c.JSON(http.StatusCreated, toTenantResponse(result))
}

// Get handles GET /api/v1/tenants/:id.
func (h *GinTenantHandler) Get(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_id", "Invalid tenant ID format", nil)
		return
	}
	result, err := h.service.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTenantResponse(result))
}

// Update handles PUT /api/v1/tenants/:id.
func (h *GinTenantHandler) Update(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_id", "Invalid tenant ID format", nil)
		return
	}
	var req UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_request", "Invalid JSON body", nil)
		return
	}
	if err := h.validator.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors[err.Field()] = getValidationMessage(err)
		}
		h.respondError(c, http.StatusBadRequest, "validation_error", "Validation failed", validationErrors)
		return
	}

	cmd := tenant.UpdateTenantCommand{
		ID:           tenantID,
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		BillingEmail: req.BillingEmail,
	}
	result, err := h.service.UpdateTenant(c.Request.Context(), cmd)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	h.logger.Info("tenant updated", zap.String("tenant_id", result.ID.String()))
	c.JSON(http.StatusOK, toTenantResponse(result))
}

// Delete handles DELETE /api/v1/tenants/:id.
func (h *GinTenantHandler) Delete(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_id", "Invalid tenant ID format", nil)
		return
	}
	if err := h.service.DeleteTenant(c.Request.Context(), tenantID); err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// List handles GET /api/v1/tenants.
func (h *GinTenantHandler) List(c *gin.Context) {
	page := parseIntQueryGin(c, "page", 1)
	pageSize := parseIntQueryGin(c, "page_size", 20)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	if page < 1 {
		page = 1
	}

	query := tenant.ListTenantsQuery{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Search:   c.Query("search"),
	}

	result, err := h.service.ListTenants(c.Request.Context(), query)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	tenants := make([]TenantResponse, len(result.Tenants))
	for i, t := range result.Tenants {
		tenants[i] = *toTenantResponse(t)
	}

	c.JSON(http.StatusOK, ListTenantsResponse{
		Tenants:    tenants,
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	})
}

// respondError sends an error response.
func (h *GinTenantHandler) respondError(c *gin.Context, status int, errCode, message string, details map[string]string) {
	c.JSON(status, ErrorResponse{
		Error:   errCode,
		Message: message,
		Details: details,
		TraceID: c.GetString("request_id"),
	})
}

// handleServiceError handles errors from the application service.
func (h *GinTenantHandler) handleServiceError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case apperrors.ErrNotFound:
			h.respondError(c, http.StatusNotFound, "not_found", appErr.Message, nil)
		case apperrors.ErrConflict:
			h.respondError(c, http.StatusConflict, "conflict", appErr.Message, nil)
		case apperrors.ErrValidation:
			h.respondError(c, http.StatusBadRequest, "validation_error", appErr.Message, nil)
		case apperrors.ErrUnauthorized:
			h.respondError(c, http.StatusUnauthorized, "unauthorized", appErr.Message, nil)
		case apperrors.ErrForbidden:
			h.respondError(c, http.StatusForbidden, "forbidden", appErr.Message, nil)
		default:
			h.logger.Error("internal error", zap.Error(err))
			h.respondError(c, http.StatusInternalServerError, "internal_error", "An internal error occurred", nil)
		}
		return
	}
	h.logger.Error("unexpected error", zap.Error(err))
	h.respondError(c, http.StatusInternalServerError, "internal_error", "An internal error occurred", nil)
}

func parseIntQueryGin(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}
