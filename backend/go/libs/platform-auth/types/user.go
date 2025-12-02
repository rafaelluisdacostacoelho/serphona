package types

import "time"

// User representa um usuário no sistema Serphona
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

// TokenResponse representa a resposta com tokens de autenticação
type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"` // segundos
}

// AuthResponse representa a resposta completa de autenticação
type AuthResponse struct {
	User   User          `json:"user"`
	Tokens TokenResponse `json:"tokens"`
}
