package middleware

import (
	"context"
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
