// Package handler contains HTTP request handlers.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	apikeyapp "tenant-manager/internal/application/apikey"
	domainapikey "tenant-manager/internal/domain/apikey"
)

// APIKeyHandler handles API key related HTTP requests.
type APIKeyHandler struct {
	service *apikeyapp.Service
	logger  *zap.Logger
}

// NewAPIKeyHandler creates a new APIKeyHandler.
func NewAPIKeyHandler(service *apikeyapp.Service, logger *zap.Logger) *APIKeyHandler {
	return &APIKeyHandler{
		service: service,
		logger:  logger,
	}
}

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

// Create handles POST /tenants/:id/api-keys
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantIDStr := chi.URLParam(r, "id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		respondAPIKeyError(w, http.StatusBadRequest, "invalid_tenant_id", "invalid tenant id")
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAPIKeyError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	if req.Name == "" || len(req.Permissions) == 0 {
		respondAPIKeyError(w, http.StatusBadRequest, "validation_error", "name and permissions are required")
		return
	}

	// TODO: get real user id from auth context
	createdBy := uuid.Nil
	key, rawKey, err := h.service.CreateAPIKey(ctx, tenantID, req.Name, createdBy, req.Permissions, req.ExpiryDays)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	resp := toAPIKeyResponse(key)
	resp.RawKey = rawKey
	writeJSON(w, http.StatusCreated, resp)
}

// List handles GET /tenants/:id/api-keys
func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantIDStr := chi.URLParam(r, "id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		respondAPIKeyError(w, http.StatusBadRequest, "invalid_tenant_id", "invalid tenant id")
		return
	}

	activeOnly := false
	if val := r.URL.Query().Get("active_only"); val != "" {
		activeOnly, _ = strconv.ParseBool(val)
	}

	keys, err := h.service.ListAPIKeys(ctx, tenantID, activeOnly)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	resp := make([]APIKeyResponse, 0, len(keys))
	for _, k := range keys {
		resp = append(resp, toAPIKeyResponse(k))
	}
	writeJSON(w, http.StatusOK, resp)
}

// Delete handles DELETE /tenants/:id/api-keys/:keyId
func (h *APIKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	keyIDStr := chi.URLParam(r, "keyId")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		respondAPIKeyError(w, http.StatusBadRequest, "invalid_key_id", "invalid key id")
		return
	}

	if err := h.service.DeleteAPIKey(ctx, keyID); err != nil {
		h.handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Gin wrapper to adapt chi-style handler to Gin router.
type GinAPIKeyHandler struct {
	handler *APIKeyHandler
}

// NewGinAPIKeyHandler creates a new Gin-compatible API key handler.
func NewGinAPIKeyHandler(service *apikeyapp.Service, logger *zap.Logger) *GinAPIKeyHandler {
	return &GinAPIKeyHandler{
		handler: NewAPIKeyHandler(service, logger),
	}
}

func (h *GinAPIKeyHandler) wrap(c *gin.Context, params map[string]string, fn func(http.ResponseWriter, *http.Request)) {
	rc := chi.NewRouteContext()
	for k, v := range params {
		rc.URLParams.Add(k, v)
	}
	ctx := context.WithValue(c.Request.Context(), chi.RouteCtxKey, rc)
	fn(c.Writer, c.Request.WithContext(ctx))
}

func (h *GinAPIKeyHandler) Create(c *gin.Context) { h.wrap(c, map[string]string{"id": c.Param("id")}, h.handler.Create) }
func (h *GinAPIKeyHandler) List(c *gin.Context)   { h.wrap(c, map[string]string{"id": c.Param("id")}, h.handler.List) }
func (h *GinAPIKeyHandler) Delete(c *gin.Context) {
	h.wrap(c, map[string]string{"id": c.Param("id"), "keyId": c.Param("keyId")}, h.handler.Delete)
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

func (h *APIKeyHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domainapikey.ErrNotFound):
		respondAPIKeyError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, domainapikey.ErrNameAlreadyExists):
		respondAPIKeyError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, domainapikey.ErrInvalidPermissions), errors.Is(err, domainapikey.ErrEmptyPermissions),
		errors.Is(err, domainapikey.ErrEmptyName), errors.Is(err, domainapikey.ErrNameTooShort), errors.Is(err, domainapikey.ErrNameTooLong):
		respondAPIKeyError(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		h.logger.Error("api key handler error", zap.Error(err))
		respondAPIKeyError(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

func respondAPIKeyError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
