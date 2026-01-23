// Package handler contains HTTP handlers.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/adapter/http/dto"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/usecase"
)

// OAuthHandler handles OAuth HTTP requests.
type OAuthHandler struct {
	oauthService usecase.OAuthService
}

// NewOAuthHandler creates a new OAuthHandler.
func NewOAuthHandler(oauthService usecase.OAuthService) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
	}
}

// Authorize handles GET /api/v1/integrations/:id/oauth/authorize.
func (h *OAuthHandler) Authorize(c *gin.Context) {
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

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_USER_ID", "user_id has invalid format", nil)
		return
	}
	integrationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid integration ID", nil)
		return
	}

	// Optional: custom scopes
	scopes := c.QueryArray("scope")

	// PKCE enabled by default for security
	usePKCE := c.Query("pkce") != "false"

	authURL, err := h.oauthService.InitiateAuthFlow(
		ctx,
		integrationID,
		tenantID,
		userID,
		scopes,
		usePKCE,
	)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "AUTH_INITIATION_FAILED", err.Error(), nil)
		return
	}

	// Return auth URL for client to redirect user
	response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.OAuthAuthorizeResponse{AuthURL: authURL})
}

// Callback handles GET /api/v1/integrations/:id/oauth/callback.
func (h *OAuthHandler) Callback(c *gin.Context) {
	ctx := c.Request.Context()
	state := c.Query("state")
	code := c.Query("code")

	if state == "" || code == "" {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "state and code are required", nil)
		return
	}

	// Handle callback
	token, err := h.oauthService.HandleCallback(ctx, state, code)
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "CALLBACK_FAILED", err.Error(), nil)
		return
	}

	var expiresAt *string
	if token.ExpiresAt != nil {
		exp := token.ExpiresAt.Format("2006-01-02T15:04:05Z")
		expiresAt = &exp
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, dto.OAuthCallbackResponse{
		TokenID:   token.ID,
		Message:   "OAuth flow completed successfully",
		ExpiresAt: expiresAt,
	})
}

// RevokeToken handles POST /api/v1/oauth/tokens/:token_id/revoke.
func (h *OAuthHandler) RevokeToken(c *gin.Context) {
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
	tokenID, err := uuid.Parse(c.Param("token_id"))
	if err != nil {
		response.WriteError(ctx, c.Writer, http.StatusBadRequest, "INVALID_ID", "invalid token ID", nil)
		return
	}

	if err := h.oauthService.RevokeToken(ctx, tenantID, tokenID); err != nil {
		response.WriteError(ctx, c.Writer, http.StatusInternalServerError, "REVOCATION_FAILED", err.Error(), nil)
		return
	}

	response.WriteSuccess(ctx, c.Writer, http.StatusOK, statusMessageResponse{Message: "token revoked successfully"})
}
