// Package handler contains HTTP request handlers.
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"tenant-manager/internal/application/tenant"
	apperrors "tenant-manager/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
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
	if !h.requireScope(c, "write:tenants") {
		return
	}

	var req CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", nil)
		return
	}
	if err := h.validator.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors[err.Field()] = getValidationMessage(err)
		}
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", validationErrors)
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
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusCreated, toTenantResponse(result))
}

// Get handles GET /api/v1/tenants/:id.
func (h *GinTenantHandler) Get(c *gin.Context) {
	if !h.requireScope(c, "read:tenants") {
		return
	}

	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_ID", "Invalid tenant ID format", nil)
		return
	}
	if !h.enforceTenantContext(c, tenantID) {
		return
	}
	result, err := h.service.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, toTenantResponse(result))
}

// Update handles PUT /api/v1/tenants/:id.
func (h *GinTenantHandler) Update(c *gin.Context) {
	if !h.requireScope(c, "write:tenants") {
		return
	}

	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_ID", "Invalid tenant ID format", nil)
		return
	}
	if !h.enforceTenantContext(c, tenantID) {
		return
	}
	var req UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", nil)
		return
	}
	if err := h.validator.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors[err.Field()] = getValidationMessage(err)
		}
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", validationErrors)
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
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, toTenantResponse(result))
}

// Delete handles DELETE /api/v1/tenants/:id.
func (h *GinTenantHandler) Delete(c *gin.Context) {
	if !h.requireScope(c, "write:tenants") {
		return
	}

	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_ID", "Invalid tenant ID format", nil)
		return
	}
	if !h.enforceTenantContext(c, tenantID) {
		return
	}
	if err := h.service.DeleteTenant(c.Request.Context(), tenantID); err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

// List handles GET /api/v1/tenants.
func (h *GinTenantHandler) List(c *gin.Context) {
	if !h.requireScope(c, "read:tenants") {
		return
	}

	tenantID, err := authmw.TenantIDFromContext(c.Request.Context())
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "missing tenant context", nil)
		return
	}

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

	// If caller is a tenant (non-platform), return only its own record.
	if tenantID != "platform" {
		tID, parseErr := uuid.Parse(tenantID)
		if parseErr != nil {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "invalid tenant context", nil)
			return
		}

		tenantDTO, getErr := h.service.GetTenant(c.Request.Context(), tID)
		if getErr != nil {
			h.handleServiceError(c, getErr)
			return
		}

		resp := ListTenantsResponse{
			Tenants:    []TenantResponse{*toTenantResponse(tenantDTO)},
			Total:      1,
			Page:       1,
			PageSize:   1,
			TotalPages: 1,
		}
		response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, resp, response.WithPagination(response.Pagination{Page: 1, PageSize: 1, Total: 1, TotalPages: 1}))
		return
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

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, ListTenantsResponse{
		Tenants:    tenants,
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}, response.WithPagination(response.Pagination{Page: result.Page, PageSize: result.PageSize, Total: int(result.Total), TotalPages: result.TotalPages}))
}

// handleServiceError handles errors from the application service.
func (h *GinTenantHandler) handleServiceError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case apperrors.ErrNotFound:
			response.WriteError(c.Request.Context(), c.Writer, http.StatusNotFound, "NOT_FOUND", appErr.Message, nil)
		case apperrors.ErrConflict:
			response.WriteError(c.Request.Context(), c.Writer, http.StatusConflict, "CONFLICT", appErr.Message, nil)
		case apperrors.ErrValidation:
			response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", appErr.Message, nil)
		case apperrors.ErrUnauthorized:
			response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", appErr.Message, nil)
		case apperrors.ErrForbidden:
			response.WriteError(c.Request.Context(), c.Writer, http.StatusForbidden, "FORBIDDEN", appErr.Message, nil)
		default:
			h.logger.Error("internal error", zap.Error(err))
			response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred", nil)
		}
		return
	}
	h.logger.Error("unexpected error", zap.Error(err))
	response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred", nil)
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

func (h *GinTenantHandler) requireScope(c *gin.Context, scope string) bool {
	claims, err := authmw.ClaimsFromContext(c.Request.Context())
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "missing claims", nil)
		return false
	}

	if !claims.HasAllScopes(scope) {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusForbidden, "FORBIDDEN", "insufficient scopes", nil)
		return false
	}

	return true
}

func (h *GinTenantHandler) enforceTenantContext(c *gin.Context, tenantID uuid.UUID) bool {
	ctxTenant, err := authmw.TenantIDFromContext(c.Request.Context())
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "missing tenant context", nil)
		return false
	}

	if ctxTenant == "platform" {
		return true
	}

	if err := authmw.EnforceTenant(authmw.WithTenantID(c.Request.Context(), ctxTenant), tenantID.String()); err != nil {
		switch {
		case errors.Is(err, autherrors.ErrUnauthorized):
			response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "missing tenant context", nil)
		case errors.Is(err, autherrors.ErrInsufficientPermissions):
			response.WriteError(c.Request.Context(), c.Writer, http.StatusForbidden, "FORBIDDEN", "tenant mismatch", nil)
		default:
			response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "tenant validation failed", nil)
		}
		return false
	}
	return true
}
