package types

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents the custom Serphona JWT claims.
type Claims struct {
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"` // user, admin, superadmin
	TenantID  string `json:"tenantId"`
	SessionID string `json:"sessionId"`
	jwt.RegisteredClaims
}

// Valid validates the custom claims fields.
func (c *Claims) Valid() error {
	if c.UserID == "" {
		return jwt.ErrTokenInvalidClaims
	}

	if _, err := uuid.Parse(c.UserID); err != nil {
		return jwt.ErrTokenInvalidClaims
	}

	if c.TenantID == "" {
		return jwt.ErrTokenInvalidClaims
	}

	if _, err := uuid.Parse(c.TenantID); err != nil {
		return jwt.ErrTokenInvalidClaims
	}

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

// HasRole returns true when the user has the provided role.
func (c *Claims) HasRole(role string) bool {
	return c.Role == role
}

// IsAdmin returns true when the user is admin or superadmin.
func (c *Claims) IsAdmin() bool {
	return c.Role == "admin" || c.Role == "superadmin"
}

// IsSuperAdmin returns true when the user is superadmin.
func (c *Claims) IsSuperAdmin() bool {
	return c.Role == "superadmin"
}
