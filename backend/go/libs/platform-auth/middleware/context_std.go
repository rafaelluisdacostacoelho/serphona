package middleware

import (
	"context"

	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

type contextKey string

const claimsContextKey contextKey = "platform-auth-claims"
const requestIDContextKey contextKey = "platform-auth-request-id"

// WithClaims attaches claims to a standard context.
func WithClaims(ctx context.Context, claims *types.Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// ClaimsFromContext retrieves claims from a standard context.
func ClaimsFromContext(ctx context.Context) (*types.Claims, error) {
	if ctx == nil {
		return nil, autherrors.ErrUnauthorized
	}

	if claims, ok := ctx.Value(claimsContextKey).(*types.Claims); ok && claims != nil {
		return claims, nil
	}

	return nil, autherrors.ErrUnauthorized
}

// WithRequestID attaches a request ID to context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// RequestIDFromContext retrieves a request ID from context.
func RequestIDFromContext(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", autherrors.ErrUnauthorized
	}

	if id, ok := ctx.Value(requestIDContextKey).(string); ok && id != "" {
		return id, nil
	}

	return "", autherrors.ErrUnauthorized
}
