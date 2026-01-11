package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"go.uber.org/zap"

	"tools-manager/internal/audit"
	"tools-manager/internal/metrics"
	"tools-manager/internal/secret"
)

// SecretHandler manages secret writes and reads.
type SecretHandler struct {
	log   *zap.Logger
	store *secret.Store
}

func NewSecretHandler(log *zap.Logger, store *secret.Store) *SecretHandler {
	return &SecretHandler{log: log, store: store}
}

type upsertSecretRequest struct {
	ID    string `json:"id" binding:"required"`
	Value string `json:"value" binding:"required"`
}

func (h *SecretHandler) Put(c *gin.Context) {
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
		return
	}

	var req upsertSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	req.ID = strings.TrimSpace(req.ID)
	if req.ID == "" {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "invalid_request", "id is required", nil)
		return
	}

	if err := h.store.Put(claims.TenantID, req.ID, req.Value); err != nil {
		h.log.Error("secret put failed", zap.Error(err))
		metrics.Errors.WithLabelValues(claims.TenantID, c.FullPath(), "500").Inc()
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "internal_error", "failed to store secret", nil)
		return
	}

	audit.Emit(h.log, audit.Event{
		Category: "secret",
		Action:   "rotate",
		Outcome:  "success",
		TenantID: claims.TenantID,
		UserID:   claims.UserID,
		Service:  claims.Service,
		Path:     c.FullPath(),
		ToolID:   c.Request.Header.Get("X-Tool-ID"),
		RuleID:   req.ID,
	})

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusCreated, gin.H{"id": req.ID})
}

func (h *SecretHandler) Get(c *gin.Context) {
	claims, err := authmw.GetClaimsFromContext(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing claims", nil)
		return
	}

	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "invalid_request", "id is required", nil)
		return
	}

	val, ok, err := h.store.Get(claims.TenantID, id)
	if err != nil {
		h.log.Error("secret get failed", zap.Error(err))
		metrics.Errors.WithLabelValues(claims.TenantID, c.FullPath(), "500").Inc()
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "internal_error", "failed to fetch secret", nil)
		return
	}
	if !ok {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusNotFound, "not_found", "secret not found", nil)
		return
	}

	audit.Emit(h.log, audit.Event{
		Category: "secret",
		Action:   "access",
		Outcome:  "success",
		TenantID: claims.TenantID,
		UserID:   claims.UserID,
		Service:  claims.Service,
		Path:     c.FullPath(),
		ToolID:   c.Request.Header.Get("X-Tool-ID"),
		RuleID:   id,
	})

	// Do not log value; return plainly to caller.
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{"id": id, "value": val})
}
