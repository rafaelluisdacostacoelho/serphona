// Package handler contains HTTP handlers.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/adapter/http/dto"
	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/usecase"
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
	integrationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "invalid integration ID",
		})
		return
	}

	tenantID, _ := c.Get("tenant_id")
	userID, _ := c.Get("user_id")

	// Optional: custom scopes
	scopes := c.QueryArray("scope")

	// PKCE enabled by default for security
	usePKCE := c.Query("pkce") != "false"

	authURL, err := h.oauthService.InitiateAuthFlow(
		c.Request.Context(),
		integrationID,
		tenantID.(uuid.UUID),
		userID.(uuid.UUID),
		scopes,
		usePKCE,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "auth_initiation_failed",
			Message: err.Error(),
		})
		return
	}

	// Return auth URL for client to redirect user
	c.JSON(http.StatusOK, dto.OAuthAuthorizeResponse{
		AuthURL: authURL,
	})
}

// Callback handles GET /api/v1/integrations/:id/oauth/callback.
func (h *OAuthHandler) Callback(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")

	if state == "" || code == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "state and code are required",
		})
		return
	}

	// Handle callback
	token, err := h.oauthService.HandleCallback(c.Request.Context(), state, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "callback_failed",
			Message: err.Error(),
		})
		return
	}

	var expiresAt *string
	if token.ExpiresAt != nil {
		exp := token.ExpiresAt.Format("2006-01-02T15:04:05Z")
		expiresAt = &exp
	}

	c.JSON(http.StatusOK, dto.OAuthCallbackResponse{
		TokenID:   token.ID,
		Message:   "OAuth flow completed successfully",
		ExpiresAt: expiresAt,
	})
}

// RevokeToken handles POST /api/v1/oauth/tokens/:token_id/revoke.
func (h *OAuthHandler) RevokeToken(c *gin.Context) {
	tokenID, err := uuid.Parse(c.Param("token_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "invalid token ID",
		})
		return
	}

	if err := h.oauthService.RevokeToken(c.Request.Context(), tokenID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "revocation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "token revoked successfully"})
}
