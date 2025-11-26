// Package handler contains HTTP request handlers.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"tenant-management/internal/application/tenant"
	apperrors "tenant-management/pkg/errors"
)

// TenantHandler handles tenant-related HTTP requests.
type TenantHandler struct {
	service   *tenant.Service
	logger    *zap.Logger
	validator *validator.Validate
}

// NewTenantHandler creates a new TenantHandler.
func NewTenantHandler(service *tenant.Service, logger *zap.Logger) *TenantHandler {
	v := validator.New()
	return &TenantHandler{
		service:   service,
		logger:    logger,
		validator: v,
	}
}

// CreateTenantRequest represents the request body for creating a tenant.
type CreateTenantRequest struct {
	Name         string `json:"name" validate:"required,min=2,max=100"`
	Email        string `json:"email" validate:"required,email"`
	Phone        string `json:"phone,omitempty" validate:"omitempty,e164"`
	Plan         string `json:"plan" validate:"required,oneof=starter professional enterprise"`
	BillingEmail string `json:"billing_email,omitempty" validate:"omitempty,email"`
	Metadata     struct {
		Industry    string `json:"industry,omitempty"`
		CompanySize string `json:"company_size,omitempty"`
		Website     string `json:"website,omitempty" validate:"omitempty,url"`
	} `json:"metadata,omitempty"`
}

// UpdateTenantRequest represents the request body for updating a tenant.
type UpdateTenantRequest struct {
	Name         *string `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Email        *string `json:"email,omitempty" validate:"omitempty,email"`
	Phone        *string `json:"phone,omitempty" validate:"omitempty,e164"`
	BillingEmail *string `json:"billing_email,omitempty" validate:"omitempty,email"`
}

// TenantResponse represents the response for tenant operations.
type TenantResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	Email        string                 `json:"email"`
	Phone        string                 `json:"phone,omitempty"`
	Status       string                 `json:"status"`
	Plan         string                 `json:"plan"`
	Settings     map[string]interface{} `json:"settings"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
	BillingEmail string                 `json:"billing_email,omitempty"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
	TraceID string            `json:"trace_id,omitempty"`
}

// ListTenantsResponse represents the response for listing tenants.
type ListTenantsResponse struct {
	Tenants    []TenantResponse `json:"tenants"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// Create handles POST /api/v1/tenants
// @Summary Create a new tenant
// @Description Creates a new tenant organization
// @Tags tenants
// @Accept json
// @Produce json
// @Param request body CreateTenantRequest true "Tenant creation request"
// @Success 201 {object} TenantResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tenants [post]
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := getRequestID(ctx)

	// Parse request body
	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body", nil)
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors[err.Field()] = getValidationMessage(err)
		}
		h.respondError(w, r, http.StatusBadRequest, "validation_error", "Validation failed", validationErrors)
		return
	}

	// Create tenant via application service
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

	result, err := h.service.CreateTenant(ctx, cmd)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	h.logger.Info("tenant created",
		zap.String("tenant_id", result.ID.String()),
		zap.String("email", result.Email),
		zap.String("request_id", requestID),
	)

	h.respondJSON(w, http.StatusCreated, toTenantResponse(result))
}

// Get handles GET /api/v1/tenants/{id}
// @Summary Get tenant by ID
// @Description Retrieves a tenant by its ID
// @Tags tenants
// @Produce json
// @Param id path string true "Tenant ID" format(uuid)
// @Success 200 {object} TenantResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tenants/{id} [get]
func (h *TenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse tenant ID from URL
	idParam := chi.URLParam(r, "id")
	tenantID, err := uuid.Parse(idParam)
	if err != nil {
		h.respondError(w, r, http.StatusBadRequest, "invalid_id", "Invalid tenant ID format", nil)
		return
	}

	// Get tenant via application service
	result, err := h.service.GetTenant(ctx, tenantID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	h.respondJSON(w, http.StatusOK, toTenantResponse(result))
}

// Update handles PUT /api/v1/tenants/{id}
// @Summary Update tenant
// @Description Updates an existing tenant
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID" format(uuid)
// @Param request body UpdateTenantRequest true "Tenant update request"
// @Success 200 {object} TenantResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tenants/{id} [put]
func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := getRequestID(ctx)

	// Parse tenant ID
	idParam := chi.URLParam(r, "id")
	tenantID, err := uuid.Parse(idParam)
	if err != nil {
		h.respondError(w, r, http.StatusBadRequest, "invalid_id", "Invalid tenant ID format", nil)
		return
	}

	// Parse request body
	var req UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body", nil)
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors[err.Field()] = getValidationMessage(err)
		}
		h.respondError(w, r, http.StatusBadRequest, "validation_error", "Validation failed", validationErrors)
		return
	}

	// Update tenant via application service
	cmd := tenant.UpdateTenantCommand{
		ID:           tenantID,
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		BillingEmail: req.BillingEmail,
	}

	result, err := h.service.UpdateTenant(ctx, cmd)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	h.logger.Info("tenant updated",
		zap.String("tenant_id", result.ID.String()),
		zap.String("request_id", requestID),
	)

	h.respondJSON(w, http.StatusOK, toTenantResponse(result))
}

// Delete handles DELETE /api/v1/tenants/{id}
// @Summary Delete tenant
// @Description Soft-deletes a tenant
// @Tags tenants
// @Produce json
// @Param id path string true "Tenant ID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tenants/{id} [delete]
func (h *TenantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := getRequestID(ctx)

	// Parse tenant ID
	idParam := chi.URLParam(r, "id")
	tenantID, err := uuid.Parse(idParam)
	if err != nil {
		h.respondError(w, r, http.StatusBadRequest, "invalid_id", "Invalid tenant ID format", nil)
		return
	}

	// Delete tenant via application service
	if err := h.service.DeleteTenant(ctx, tenantID); err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	h.logger.Info("tenant deleted",
		zap.String("tenant_id", tenantID.String()),
		zap.String("request_id", requestID),
	)

	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /api/v1/tenants
// @Summary List tenants
// @Description Lists tenants with pagination
// @Tags tenants
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param status query string false "Filter by status"
// @Param search query string false "Search in name/email"
// @Success 200 {object} ListTenantsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tenants [get]
func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	query := tenant.ListTenantsQuery{
		Page:     parseIntQuery(r, "page", 1),
		PageSize: parseIntQuery(r, "page_size", 20),
		Status:   r.URL.Query().Get("status"),
		Search:   r.URL.Query().Get("search"),
	}

	// Validate pagination
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}
	if query.Page < 1 {
		query.Page = 1
	}

	// List tenants via application service
	result, err := h.service.ListTenants(ctx, query)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	// Build response
	tenants := make([]TenantResponse, len(result.Tenants))
	for i, t := range result.Tenants {
		tenants[i] = *toTenantResponse(t)
	}

	response := ListTenantsResponse{
		Tenants:    tenants,
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// respondJSON sends a JSON response.
func (h *TenantHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// respondError sends an error response.
func (h *TenantHandler) respondError(w http.ResponseWriter, r *http.Request, status int, errCode, message string, details map[string]string) {
	response := ErrorResponse{
		Error:   errCode,
		Message: message,
		Details: details,
		TraceID: getRequestID(r.Context()),
	}
	h.respondJSON(w, status, response)
}

// handleServiceError handles errors from the application service.
func (h *TenantHandler) handleServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case apperrors.ErrNotFound:
			h.respondError(w, r, http.StatusNotFound, "not_found", appErr.Message, nil)
		case apperrors.ErrConflict:
			h.respondError(w, r, http.StatusConflict, "conflict", appErr.Message, nil)
		case apperrors.ErrValidation:
			h.respondError(w, r, http.StatusBadRequest, "validation_error", appErr.Message, nil)
		case apperrors.ErrUnauthorized:
			h.respondError(w, r, http.StatusUnauthorized, "unauthorized", appErr.Message, nil)
		case apperrors.ErrForbidden:
			h.respondError(w, r, http.StatusForbidden, "forbidden", appErr.Message, nil)
		default:
			h.logger.Error("internal error", zap.Error(err))
			h.respondError(w, r, http.StatusInternalServerError, "internal_error", "An internal error occurred", nil)
		}
		return
	}

	h.logger.Error("unexpected error", zap.Error(err))
	h.respondError(w, r, http.StatusInternalServerError, "internal_error", "An internal error occurred", nil)
}

// toTenantResponse converts a domain tenant to a response DTO.
func toTenantResponse(t *tenant.TenantDTO) *TenantResponse {
	return &TenantResponse{
		ID:           t.ID.String(),
		Name:         t.Name,
		Slug:         t.Slug,
		Email:        t.Email,
		Phone:        t.Phone,
		Status:       t.Status,
		Plan:         t.Plan,
		CreatedAt:    t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		BillingEmail: t.BillingEmail,
	}
}

// Helper functions

func getRequestID(ctx context.Context) string {
	if id, ok := ctx.Value("request_id").(string); ok {
		return id
	}
	return ""
}

func parseIntQuery(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	var result int
	if _, err := fmt.Sscanf(val, "%d", &result); err != nil {
		return defaultVal
	}
	return result
}

func getValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return "Value is too short"
	case "max":
		return "Value is too long"
	case "oneof":
		return "Invalid value"
	case "url":
		return "Invalid URL format"
	case "e164":
		return "Invalid phone number format"
	default:
		return "Invalid value"
	}
}
