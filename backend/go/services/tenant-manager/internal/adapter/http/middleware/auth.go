// Package middleware provides HTTP middleware implementations.
package middleware

import (
	"net/http"
)

// AuthMiddleware handles JWT authentication.
type AuthMiddleware struct {
	secret string
	issuer string
}

// NewAuthMiddleware creates a new auth middleware.
func NewAuthMiddleware(secret, issuer string) *AuthMiddleware {
	return &AuthMiddleware{
		secret: secret,
		issuer: issuer,
	}
}

// Handle is the middleware handler function.
func (m *AuthMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For now, just pass through - implement JWT validation here
		next.ServeHTTP(w, r)
	})
}
