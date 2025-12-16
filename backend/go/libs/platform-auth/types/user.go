package types

import "time"

// User represents a user in the Serphona platform.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	TenantID  string    `json:"tenantId"`
	Provider  string    `json:"provider"` // local, google, microsoft, apple
	Verified  bool      `json:"verified"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TokenResponse represents the authentication token payload.
type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"` // seconds
}

// AuthResponse represents the full authentication response.
type AuthResponse struct {
	User   User          `json:"user"`
	Tokens TokenResponse `json:"tokens"`
}
