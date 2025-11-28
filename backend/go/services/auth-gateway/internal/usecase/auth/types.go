package auth

import "github.com/google/uuid"

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	Name       string `json:"name" validate:"required"`
	TenantName string `json:"tenantName" validate:"required"`
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User   UserResponse   `json:"user"`
	Tokens TokensResponse `json:"tokens"`
}

// UserResponse represents a user in the response
type UserResponse struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	TenantID uuid.UUID `json:"tenantId"`
}

// TokensResponse represents authentication tokens
type TokensResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"` // seconds
}

// OAuthCallbackRequest represents OAuth callback data
type OAuthCallbackRequest struct {
	Code  string
	State string
}

// OAuthURLResponse represents OAuth authorization URL
type OAuthURLResponse struct {
	URL string `json:"url"`
}
