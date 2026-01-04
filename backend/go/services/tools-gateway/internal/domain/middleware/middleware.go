package middleware

import (
	"context"
	"errors"
)

const TenantIDHeader = "X-Tenant-ID"

type tenantKey struct{}

// WithTenantID adds the tenant ID to the context
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey{}, tenantID)
}

// TenantIDFromContext retrieves the tenant ID from the context
func TenantIDFromContext(ctx context.Context) (string, error) {
	if tenantID, ok := ctx.Value(tenantKey{}).(string); ok {
		return tenantID, nil
	}
	return "", errors.New("tenant ID not found in context")
}
