// Package handler contains HTTP request handlers.
package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"go.uber.org/zap"

	apikeyapp "tenant-manager/internal/application/apikey"
	domainapikey "tenant-manager/internal/domain/apikey"
)

// CreateAPIKeyRequest represents the body for creating an API key.
type CreateAPIKeyRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	ExpiryDays  int      `json:"expiry_days,omitempty"`
	Description string   `json:"description,omitempty"`
}

// APIKeyResponse represents the API key response (with raw key on creation).
type APIKeyResponse struct {
	ID          string   `json:"id"`
	TenantID    string   `json:"tenant_id"`
	Name        string   `json:"name"`
	KeyPrefix   string   `json:"key_prefix"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
	ExpiresAt   *string  `json:"expires_at,omitempty"`
	CreatedAt   string   `json:"created_at"`
	LastUsedAt  *string  `json:"last_used_at,omitempty"`
	RawKey      string   `json:"raw_key,omitempty"` // Only returned on create
}

// GinAPIKeyHandler handles API key related HTTP requests using Gin.
type GinAPIKeyHandler struct {
	service *apikeyapp.Service
	logger  *zap.Logger
}

// NewGinAPIKeyHandler creates a new Gin-compatible API key handler.
func NewGinAPIKeyHandler(service *apikeyapp.Service, logger *zap.Logger) *GinAPIKeyHandler {
	return &GinAPIKeyHandler{
		service: service,
		logger:  logger,
	}
}

// Create handles POST /tenants/:id/api-keys
func (h *GinAPIKeyHandler) Create(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_TENANT_ID", "invalid tenant id", nil)
		return
	}
	if !h.enforceTenantContext(c, tenantID) {
		return
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body", nil)
		return
	}
	if req.Name == "" || len(req.Permissions) == 0 {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", "name and permissions are required", nil)
		return
	}

	key, rawKey, err := h.service.CreateAPIKey(c.Request.Context(), tenantID, req.Name, uuid.Nil, req.Permissions, req.ExpiryDays)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	resp := toAPIKeyResponse(key)
	resp.RawKey = rawKey
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusCreated, resp)
}

// List handles GET /tenants/:id/api-keys
func (h *GinAPIKeyHandler) List(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_TENANT_ID", "invalid tenant id", nil)
		return
	}
	if !h.enforceTenantContext(c, tenantID) {
		return
	}

	activeOnly := false
	if val := c.Query("active_only"); val != "" {
		activeOnly, _ = strconv.ParseBool(val)
	}

	keys, err := h.service.ListAPIKeys(c.Request.Context(), tenantID, activeOnly)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	resp := make([]APIKeyResponse, 0, len(keys))
	for _, k := range keys {
		resp = append(resp, toAPIKeyResponse(k))
	}
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, resp)
}

// Delete handles DELETE /tenants/:id/api-keys/:keyId
func (h *GinAPIKeyHandler) Delete(c *gin.Context) {
	keyID, err := uuid.Parse(c.Param("keyId"))
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_KEY_ID", "invalid key id", nil)
		return
	}

	if err := h.service.DeleteAPIKey(c.Request.Context(), keyID); err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusNoContent, nil)
}

// Helpers
func toAPIKeyResponse(k *domainapikey.APIKey) APIKeyResponse {
	var expiresAt *string
	if k.ExpiresAt != nil {
		s := k.ExpiresAt.UTC().Format(time.RFC3339)
		expiresAt = &s
	}
	var lastUsed *string
	if k.LastUsedAt != nil {
		s := k.LastUsedAt.UTC().Format(time.RFC3339)
		lastUsed = &s
	}
	return APIKeyResponse{
		ID:          k.ID.String(),
		TenantID:    k.TenantID.String(),
		Name:        k.Name,
		KeyPrefix:   k.KeyPrefix,
		Permissions: k.Permissions,
		Status:      string(k.Status),
		ExpiresAt:   expiresAt,
		CreatedAt:   k.CreatedAt.UTC().Format(time.RFC3339),
		LastUsedAt:  lastUsed,
	}
}

func (h *GinAPIKeyHandler) handleDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, autherrors.ErrUnauthorized):
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "missing tenant context", nil)
	case errors.Is(err, autherrors.ErrInsufficientPermissions):
		response.WriteError(c.Request.Context(), c.Writer, http.StatusForbidden, "FORBIDDEN", "tenant mismatch", nil)
	case errors.Is(err, domainapikey.ErrNotFound):
		response.WriteError(c.Request.Context(), c.Writer, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, domainapikey.ErrNameAlreadyExists):
		response.WriteError(c.Request.Context(), c.Writer, http.StatusConflict, "CONFLICT", err.Error(), nil)
	case errors.Is(err, domainapikey.ErrInvalidPermissions), errors.Is(err, domainapikey.ErrEmptyPermissions),
		errors.Is(err, domainapikey.ErrEmptyName), errors.Is(err, domainapikey.ErrNameTooShort), errors.Is(err, domainapikey.ErrNameTooLong):
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	default:
		h.logger.Error("api key handler error", zap.Error(err))
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error", nil)
	}
}

func (h *GinAPIKeyHandler) enforceTenantContext(c *gin.Context, tenantID uuid.UUID) bool {
	if err := authmw.EnforceTenant(c.Request.Context(), tenantID.String()); err != nil {
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
