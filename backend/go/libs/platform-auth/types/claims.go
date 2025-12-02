package types

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims representa as claims customizadas do JWT do Serphona
type Claims struct {
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"` // user, admin, superadmin
	TenantID  string `json:"tenantId"`
	SessionID string `json:"sessionId"`
	jwt.RegisteredClaims
}

// Valid valida as claims customizadas
func (c *Claims) Valid() error {
	// Validar UserID
	if c.UserID == "" {
		return jwt.ErrTokenInvalidClaims
	}

	// Validar UUID format
	if _, err := uuid.Parse(c.UserID); err != nil {
		return jwt.ErrTokenInvalidClaims
	}

	// Validar TenantID
	if c.TenantID == "" {
		return jwt.ErrTokenInvalidClaims
	}

	if _, err := uuid.Parse(c.TenantID); err != nil {
		return jwt.ErrTokenInvalidClaims
	}

	// Validar Role
	validRoles := map[string]bool{
		"user":       true,
		"admin":      true,
		"superadmin": true,
	}

	if !validRoles[c.Role] {
		return jwt.ErrTokenInvalidClaims
	}

	return nil
}

// HasRole verifica se o usuário tem uma role específica
func (c *Claims) HasRole(role string) bool {
	return c.Role == role
}

// IsAdmin verifica se o usuário é admin ou superadmin
func (c *Claims) IsAdmin() bool {
	return c.Role == "admin" || c.Role == "superadmin"
}

// IsSuperAdmin verifica se o usuário é superadmin
func (c *Claims) IsSuperAdmin() bool {
	return c.Role == "superadmin"
}
