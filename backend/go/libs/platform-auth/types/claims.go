package types

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents the custom Serphona JWT claims.
type Claims struct {
	UserID    string   `json:"userId"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Role      string   `json:"role"` // user, admin, superadmin
	TenantID  string   `json:"tenantId"`
	SessionID string   `json:"sessionId"`
	Service   string   `json:"service,omitempty"`
	Scopes    []string `json:"scopes,omitempty"` // Optional fine-grained permissions
	jwt.RegisteredClaims
}

// Valid validates the custom claims fields.
func (c Claims) Valid() error {
	tenant := strings.TrimSpace(c.TenantID)
	if tenant == "" {
		return jwt.ErrTokenInvalidClaims
	}

	if tenant != "platform" {
		if _, err := uuid.Parse(tenant); err != nil {
			return jwt.ErrTokenInvalidClaims
		}
	}

	hasService := strings.TrimSpace(c.Service) != ""
	hasUser := strings.TrimSpace(c.UserID) != ""

	if hasService && hasUser {
		return jwt.ErrTokenInvalidClaims
	}

	if !hasService && !hasUser {
		return jwt.ErrTokenInvalidClaims
	}

	if hasUser {
		if _, err := uuid.Parse(strings.TrimSpace(c.UserID)); err != nil {
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

// HasScope returns true when the user has the specified scope.
func (c *Claims) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasAnyScope returns true when the user has at least one of the specified scopes.
func (c *Claims) HasAnyScope(scopes ...string) bool {
	for _, required := range scopes {
		if c.HasScope(required) {
			return true
		}
	}
	return false
}

// HasAllScopes returns true when the user has all of the specified scopes.
func (c *Claims) HasAllScopes(scopes ...string) bool {
	for _, required := range scopes {
		if !c.HasScope(required) {
			return false
		}
	}
	return true
}
