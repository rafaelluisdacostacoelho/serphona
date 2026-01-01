package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
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
	var req auth.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", formatValidationErrors(err))
		return
	}

	resp, err := h.authUC.Register(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

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
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", formatValidationErrors(err))
		return
	}

	resp, err := h.authUC.Login(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

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
	var req auth.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	resp, err := h.authUC.RefreshToken(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

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
	userID, exists := c.Get("userID")
	if !exists {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	if err := h.authUC.Logout(c.Request.Context(), userID.(uuid.UUID)); err != nil {
		h.handleError(c, err)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusNoContent, nil)
}

// GetOAuthURL generates OAuth authorization URL
// @Summary Get OAuth authorization URL
// @Tags OAuth
// @Param provider path string true "OAuth provider (google, microsoft, apple)"
// @Produce json
// @Success 200 {object} auth.OAuthURLResponse
// @Failure 400 {object} ErrorResponse
// @Router /auth/oauth/{provider} [get]
func (h *AuthHandler) GetOAuthURL(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "Provider is required", nil)
		return
	}

	resp, err := h.authUC.GetOAuthURL(c.Request.Context(), provider)
	if err != nil {
		h.handleError(c, err)
		return
	}

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
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "Missing code or state", nil)
		return
	}

	req := auth.OAuthCallbackRequest{
		Code:  code,
		State: state,
	}

	resp, err := h.authUC.HandleOAuthCallback(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

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
		h.logger.Error("Internal server error", zap.Error(err))
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
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
