package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/domain/user"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/service/jwt"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrSessionNotFound    = errors.New("session not found")
)

// UseCase handles authentication business logic
type UseCase struct {
	userRepo          user.Repository
	jwtService        *jwt.Service
	tenantService     TenantService
	oauthProviders    map[string]OAuthProvider
	accessTokenExpiry time.Duration
}

// TenantService defines tenant management operations
type TenantService interface {
	CreateTenant(ctx context.Context, name string) (uuid.UUID, error)
}

// OAuthProvider defines OAuth provider interface
type OAuthProvider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*OAuthUserInfo, error)
}

// OAuthUserInfo represents user info from OAuth provider
type OAuthUserInfo struct {
	ProviderID string
	Email      string
	Name       string
	Verified   bool
}

// NewUseCase creates a new authentication use case
func NewUseCase(
	userRepo user.Repository,
	jwtService *jwt.Service,
	tenantService TenantService,
	accessTokenExpiry time.Duration,
) *UseCase {
	return &UseCase{
		userRepo:          userRepo,
		jwtService:        jwtService,
		tenantService:     tenantService,
		oauthProviders:    make(map[string]OAuthProvider),
		accessTokenExpiry: accessTokenExpiry,
	}
}

// RegisterOAuthProvider registers an OAuth provider
func (uc *UseCase) RegisterOAuthProvider(name string, provider OAuthProvider) {
	uc.oauthProviders[name] = provider
}

// Register registers a new user
func (uc *UseCase) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	// Check if email already exists
	existingUser, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Create tenant
	tenantID, err := uc.tenantService.CreateTenant(ctx, req.TenantName)
	if err != nil {
		return nil, err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	newUser := &user.User{
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
		TenantID: tenantID,
		Role:     "user",
		Provider: "local",
		Verified: false,
		Active:   true,
	}

	if err := uc.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}

	// Generate tokens
	return uc.generateAuthResponse(ctx, newUser)
}

// Login authenticates a user
func (uc *UseCase) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	// Get user by email
	u, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !u.Active {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	return uc.generateAuthResponse(ctx, u)
}

// RefreshToken generates new tokens using refresh token
func (uc *UseCase) RefreshToken(ctx context.Context, req RefreshTokenRequest) (*AuthResponse, error) {
	// Validate refresh token
	userID, err := uc.jwtService.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Get session
	session, err := uc.userRepo.GetSession(ctx, req.RefreshToken)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	// Check if session is revoked or expired
	if session.RevokedAt != nil || session.ExpiresAt.Before(time.Now()) {
		return nil, ErrInvalidToken
	}

	// Get user
	u, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !u.Active {
		return nil, ErrInvalidCredentials
	}

	// Revoke old session
	if err := uc.userRepo.RevokeSession(ctx, req.RefreshToken); err != nil {
		return nil, err
	}

	// Generate new tokens
	return uc.generateAuthResponse(ctx, u)
}

// GetCurrentUser retrieves the current authenticated user
func (uc *UseCase) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	u, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &UserResponse{
		ID:       u.ID,
		Email:    u.Email,
		Name:     u.Name,
		Role:     u.Role,
		TenantID: u.TenantID,
	}, nil
}

// Logout revokes all sessions for a user
func (uc *UseCase) Logout(ctx context.Context, userID uuid.UUID) error {
	return uc.userRepo.RevokeAllUserSessions(ctx, userID)
}

// GetOAuthURL generates OAuth authorization URL
func (uc *UseCase) GetOAuthURL(ctx context.Context, provider string) (*OAuthURLResponse, error) {
	oauthProvider, ok := uc.oauthProviders[provider]
	if !ok {
		return nil, errors.New("provider not supported")
	}

	// Generate state
	state, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	// Store state
	oauthState := &user.OAuthState{
		State:     state,
		Provider:  provider,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	if err := uc.userRepo.CreateOAuthState(ctx, oauthState); err != nil {
		return nil, err
	}

	url := oauthProvider.GetAuthURL(state)
	return &OAuthURLResponse{URL: url}, nil
}

// HandleOAuthCallback handles OAuth callback
func (uc *UseCase) HandleOAuthCallback(ctx context.Context, req OAuthCallbackRequest) (*AuthResponse, error) {
	// Verify state
	oauthState, err := uc.userRepo.GetOAuthState(ctx, req.State)
	if err != nil {
		return nil, errors.New("invalid state")
	}

	if oauthState.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("state expired")
	}

	// Delete used state
	defer uc.userRepo.DeleteOAuthState(ctx, req.State)

	// Get provider
	oauthProvider, ok := uc.oauthProviders[oauthState.Provider]
	if !ok {
		return nil, errors.New("provider not found")
	}

	// Exchange code for user info
	userInfo, err := oauthProvider.ExchangeCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}

	// Try to find existing user
	u, err := uc.userRepo.GetByProvider(ctx, oauthState.Provider, userInfo.ProviderID)
	if err != nil {
		// User doesn't exist, try by email
		u, err = uc.userRepo.GetByEmail(ctx, userInfo.Email)
		if err != nil {
			// Create new user with new tenant
			tenantID, err := uc.tenantService.CreateTenant(ctx, userInfo.Name+"'s Organization")
			if err != nil {
				return nil, err
			}

			u = &user.User{
				Email:      userInfo.Email,
				Name:       userInfo.Name,
				TenantID:   tenantID,
				Role:       "user",
				Provider:   oauthState.Provider,
				ProviderID: userInfo.ProviderID,
				Verified:   userInfo.Verified,
				Active:     true,
				Password:   "", // No password for OAuth users
			}

			if err := uc.userRepo.Create(ctx, u); err != nil {
				return nil, err
			}
		} else {
			// Link OAuth to existing user
			u.Provider = oauthState.Provider
			u.ProviderID = userInfo.ProviderID
			if err := uc.userRepo.Update(ctx, u); err != nil {
				return nil, err
			}
		}
	}

	if !u.Active {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	return uc.generateAuthResponse(ctx, u)
}

// generateAuthResponse creates an auth response with tokens
func (uc *UseCase) generateAuthResponse(ctx context.Context, u *user.User) (*AuthResponse, error) {
	// Generate access token
	accessToken, err := uc.jwtService.GenerateAccessToken(u.ID, u.TenantID, u.Email, u.Role)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := uc.jwtService.GenerateRefreshToken(u.ID)
	if err != nil {
		return nil, err
	}

	// Create session
	session := &user.Session{
		UserID:       u.ID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt:    time.Now(),
	}

	if err := uc.userRepo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return &AuthResponse{
		User: UserResponse{
			ID:       u.ID,
			Email:    u.Email,
			Name:     u.Name,
			Role:     u.Role,
			TenantID: u.TenantID,
		},
		Tokens: TokensResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    int(uc.accessTokenExpiry.Seconds()),
		},
	}, nil
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
