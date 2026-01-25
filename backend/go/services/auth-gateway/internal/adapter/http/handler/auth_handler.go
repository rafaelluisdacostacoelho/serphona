package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/observability"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/service/jwt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/usecase/auth"
	"go.uber.org/zap"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authUC    *auth.UseCase
	jwtSvc    *jwt.Service
	validator *validator.Validate
	logger    *zap.Logger
}

// ErrorResponse documents the error envelope returned by the API.
type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authUC *auth.UseCase, jwtSvc *jwt.Service, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authUC:    authUC,
		jwtSvc:    jwtSvc,
		validator: validator.New(),
		logger:    logger,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.RegisterRequest true "Registration data"
// @Success 201 {object} auth.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	start := time.Now()
	var req auth.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.auditFailure(c, "register", err, map[string]any{"reason": "invalid_request"}, start)
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		h.auditFailure(c, "register", err, map[string]any{"reason": "validation_error"}, start)
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", formatValidationErrors(err))
		return
	}

	resp, err := h.authUC.Register(c.Request.Context(), req)
	if err != nil {
		h.auditFailure(c, "register", err, map[string]any{
			"email":  observability.MaskEmail(req.Email),
			"reason": h.errorReason(err),
		}, start)
		h.handleError(c, err)
		return
	}

	h.auditSuccess(c, "register", map[string]any{
		"user_id":   resp.User.ID,
		"tenant_id": resp.User.TenantID,
		"email":     observability.MaskEmail(resp.User.Email),
	}, start)
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusCreated, resp)
}

// Login handles user login
// @Summary Login user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "Login credentials"
// @Success 200 {object} auth.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	start := time.Now()
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.auditFailure(c, "login", err, map[string]any{"reason": "invalid_request"}, start)
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		h.auditFailure(c, "login", err, map[string]any{"reason": "validation_error"}, start)
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", formatValidationErrors(err))
		return
	}

	resp, err := h.authUC.Login(c.Request.Context(), req)
	if err != nil {
		h.auditFailure(c, "login", err, map[string]any{
			"email":  observability.MaskEmail(req.Email),
			"reason": h.errorReason(err),
		}, start)
		h.handleError(c, err)
		return
	}

	h.auditSuccess(c, "login", map[string]any{
		"user_id":   resp.User.ID,
		"tenant_id": resp.User.TenantID,
		"email":     observability.MaskEmail(resp.User.Email),
	}, start)
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, resp)
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} auth.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	start := time.Now()
	var req auth.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.auditFailure(c, "refresh", err, map[string]any{"reason": "invalid_request"}, start)
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	resp, err := h.authUC.RefreshToken(c.Request.Context(), req)
	if err != nil {
		h.auditFailure(c, "refresh", err, map[string]any{"reason": h.errorReason(err)}, start)
		h.handleError(c, err)
		return
	}

	h.auditSuccess(c, "refresh", map[string]any{
		"user_id":   resp.User.ID,
		"tenant_id": resp.User.TenantID,
	}, start)
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, resp)
}

// GetCurrentUser returns the current authenticated user
// @Summary Get current user
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} auth.UserResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	user, err := h.authUC.GetCurrentUser(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, user)
}

// Logout handles user logout
// @Summary Logout user
// @Tags Auth
// @Security BearerAuth
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	start := time.Now()
	userID, exists := c.Get("userID")
	if !exists {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	if err := h.authUC.Logout(c.Request.Context(), userID.(uuid.UUID)); err != nil {
		h.auditFailure(c, "logout", err, map[string]any{"reason": h.errorReason(err)}, start)
		h.handleError(c, err)
		return
	}

	tenantID, _ := c.Get("tenantID")
	h.auditSuccess(c, "logout", map[string]any{
		"user_id":   userID,
		"tenant_id": tenantID,
	}, start)
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusNoContent, nil)
}

// GetOAuthURL generates OAuth authorization URL
// @Summary Get OAuth authorization URL
// @Tags OAuth
// @Param provider path string true "OAuth provider (google, microsoft)"
// @Produce json
// @Success 200 {object} auth.OAuthURLResponse
// @Failure 400 {object} ErrorResponse
// @Router /auth/oauth/{provider} [get]
func (h *AuthHandler) GetOAuthURL(c *gin.Context) {
	start := time.Now()
	provider := c.Param("provider")
	if provider == "" {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "Provider is required", nil)
		return
	}

	resp, err := h.authUC.GetOAuthURL(c.Request.Context(), provider)
	if err != nil {
		h.auditFailure(c, "oauth_url", err, map[string]any{"provider": provider, "reason": h.errorReason(err)}, start)
		h.handleError(c, err)
		return
	}

	h.auditSuccess(c, "oauth_url", map[string]any{"provider": provider}, start)
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, resp)
}

// HandleOAuthCallback handles OAuth provider callback
// @Summary Handle OAuth callback
// @Tags OAuth
// @Param provider path string true "OAuth provider"
// @Param code query string true "Authorization code"
// @Param state query string true "State parameter"
// @Produce json
// @Success 200 {object} auth.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Router /auth/oauth/{provider}/callback [get]
func (h *AuthHandler) HandleOAuthCallback(c *gin.Context) {
	start := time.Now()
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		h.auditFailure(c, "oauth_callback", auth.ErrInvalidState, map[string]any{
			"reason":   "invalid_request",
			"provider": c.Param("provider"),
		}, start)
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "Missing code or state", nil)
		return
	}

	req := auth.OAuthCallbackRequest{
		Code:  code,
		State: state,
	}

	resp, err := h.authUC.HandleOAuthCallback(c.Request.Context(), req)
	if err != nil {
		h.auditFailure(c, "oauth_callback", err, map[string]any{
			"provider": c.Param("provider"),
			"reason":   h.errorReason(err),
		}, start)
		h.handleError(c, err)
		return
	}

	h.auditSuccess(c, "oauth_callback", map[string]any{
		"provider":  c.Param("provider"),
		"user_id":   resp.User.ID,
		"tenant_id": resp.User.TenantID,
		"email":     observability.MaskEmail(resp.User.Email),
	}, start)
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, resp)
}

// handleError handles use case errors and converts them to HTTP responses
func (h *AuthHandler) handleError(c *gin.Context, err error) {
	switch err {
	case auth.ErrInvalidCredentials:
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials", nil)
	case auth.ErrUserNotFound:
		response.WriteError(c.Request.Context(), c.Writer, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil)
	case auth.ErrEmailAlreadyExists:
		response.WriteError(c.Request.Context(), c.Writer, http.StatusConflict, "EMAIL_EXISTS", "Email already exists", nil)
	case auth.ErrInvalidToken, auth.ErrSessionNotFound:
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token", nil)
	case auth.ErrUnsupportedProvider:
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "UNSUPPORTED_PROVIDER", "Provider not supported", nil)
	case auth.ErrInvalidState, auth.ErrStateExpired:
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_STATE", "Invalid or expired OAuth state", nil)
	default:
		if reqID := c.GetString("request_id"); reqID != "" {
			h.logger.Error("Internal server error", zap.Error(err), zap.String("request_id", reqID))
		} else {
			h.logger.Error("Internal server error", zap.Error(err))
		}
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
	}
}

func (h *AuthHandler) auditSuccess(c *gin.Context, event string, details map[string]any, start time.Time) {
	observability.RecordAuthEvent(c.Request.Context(), h.logger, event, "success", details)
	if !start.IsZero() {
		observability.ObserveAuthLatency(event, "success", time.Since(start))
	}
}

func (h *AuthHandler) auditFailure(c *gin.Context, event string, err error, details map[string]any, start time.Time) {
	if details == nil {
		details = map[string]any{}
	}
	if details["reason"] == nil {
		details["reason"] = h.errorReason(err)
	}
	observability.RecordAuthEvent(c.Request.Context(), h.logger, event, "failure", details)
	if !start.IsZero() {
		observability.ObserveAuthLatency(event, "failure", time.Since(start))
	}
}

func (h *AuthHandler) errorReason(err error) string {
	switch err {
	case auth.ErrInvalidCredentials:
		return "invalid_credentials"
	case auth.ErrUserNotFound:
		return "user_not_found"
	case auth.ErrEmailAlreadyExists:
		return "email_exists"
	case auth.ErrInvalidToken, auth.ErrSessionNotFound:
		return "invalid_token"
	case auth.ErrUnsupportedProvider:
		return "unsupported_provider"
	case auth.ErrInvalidState, auth.ErrStateExpired:
		return "invalid_state"
	default:
		return "internal_error"
	}
}

// formatValidationErrors formats validator errors
func formatValidationErrors(err error) map[string]interface{} {
	details := make(map[string]interface{})
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			details[e.Field()] = e.Tag()
		}
	}
	return details
}
