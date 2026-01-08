package middleware

import (
	"context"
	"fmt"
	"net/http"

	platformmiddleware "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

const TenantIDHeader = platformmiddleware.TenantIDHeader

// WithTenantID adds the tenant ID to the context using the shared platform helper.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return platformmiddleware.WithTenantID(ctx, tenantID)
}

// TenantIDFromContext retrieves the tenant ID from the context.
func TenantIDFromContext(ctx context.Context) (string, error) {
	return platformmiddleware.TenantIDFromContext(ctx)
}

// EnsureTenantHeader clones headers and sets X-Tenant-Id when missing.
func EnsureTenantHeader(headers http.Header, tenantID string) http.Header {
	return platformmiddleware.EnsureTenantHeader(headers, tenantID)
}

// WithScopes stores scopes in context for downstream propagation.
func WithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, scopesKey{}, scopes)
}

// ScopesFromContext extracts scopes when present.
func ScopesFromContext(ctx context.Context) ([]string, error) {
	val := ctx.Value(scopesKey{})
	if val == nil {
		return nil, fmt.Errorf("scopes not found in context")
	}
	scopes, ok := val.([]string)
	if !ok {
		return nil, fmt.Errorf("scopes context value has invalid type")
	}
	return scopes, nil
}

type scopesKey struct{}
