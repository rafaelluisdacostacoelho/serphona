package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/service/jwt"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/usecase/auth"
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
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
			Code:    "INVALID_REQUEST",
		})
		return
	}

	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Validation failed",
			Code:    "VALIDATION_ERROR",
			Details: formatValidationErrors(err),
		})
		return
	}

	resp, err := h.authUC.Register(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
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
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
			Code:    "INVALID_REQUEST",
		})
		return
	}

	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Validation failed",
			Code:    "VALIDATION_ERROR",
			Details: formatValidationErrors(err),
		})
		return
	}

	resp, err := h.authUC.Login(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
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
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
			Code:    "INVALID_REQUEST",
		})
		return
	}

	resp, err := h.authUC.RefreshToken(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
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
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Message: "Unauthorized",
			Code:    "UNAUTHORIZED",
		})
		return
	}

	user, err := h.authUC.GetCurrentUser(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
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
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Message: "Unauthorized",
			Code:    "UNAUTHORIZED",
		})
		return
	}

	if err := h.authUC.Logout(c.Request.Context(), userID.(uuid.UUID)); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
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
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Provider is required",
			Code:    "INVALID_REQUEST",
		})
		return
	}

	resp, err := h.authUC.GetOAuthURL(c.Request.Context(), provider)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
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
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Missing code or state",
			Code:    "INVALID_REQUEST",
		})
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

	c.JSON(http.StatusOK, resp)
}

// handleError handles use case errors and converts them to HTTP responses
func (h *AuthHandler) handleError(c *gin.Context, err error) {
	switch err {
	case auth.ErrInvalidCredentials:
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Message: "Invalid credentials",
			Code:    "INVALID_CREDENTIALS",
		})
	case auth.ErrUserNotFound:
		c.JSON(http.StatusNotFound, ErrorResponse{
			Message: "User not found",
			Code:    "USER_NOT_FOUND",
		})
	case auth.ErrEmailAlreadyExists:
		c.JSON(http.StatusConflict, ErrorResponse{
			Message: "Email already exists",
			Code:    "EMAIL_EXISTS",
		})
	case auth.ErrInvalidToken, auth.ErrSessionNotFound:
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Message: "Invalid or expired token",
			Code:    "INVALID_TOKEN",
		})
	default:
		h.logger.Error("Internal server error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "Internal server error",
			Code:    "INTERNAL_ERROR",
		})
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Message string                 `json:"message"`
	Code    string                 `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
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
