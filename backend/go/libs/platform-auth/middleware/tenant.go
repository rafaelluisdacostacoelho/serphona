package middleware

import (
	"context"
	"net/http"

	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
)

const TenantIDHeader = "X-Tenant-Id"
const tenantContextKey contextKey = "platform-auth-tenant-id"
const unknownTenantLabel = "unknown"

func tenantFromHeaders(headers http.Header) string {
	if headers == nil {
		return unknownTenantLabel
	}
	if tenant := headers.Get(TenantIDHeader); tenant != "" {
		return tenant
	}
	return unknownTenantLabel
}

// WithTenantID attaches a tenant id to context for downstream DB/Kafka helpers.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithValue(ctx, tenantContextKey, tenantID)
}

// TenantIDFromContext resolves tenant id from claims or explicit tenant context.
func TenantIDFromContext(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Prioridade: tenant explícito no contexto
	if tenant, ok := ctx.Value(tenantContextKey).(string); ok && tenant != "" {
		return tenant, nil
	}

	// Verifica as claims
	if claims, err := ClaimsFromContext(ctx); err == nil && claims.TenantID != "" {
		return claims.TenantID, nil
	}

	return "", autherrors.ErrUnauthorized
}

// EnforceTenant ensures the provided tenant matches what is present in context claims/tenant value.
func EnforceTenant(ctx context.Context, tenantID string) error {
	ctxTenant, err := TenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	if tenantID != "" && tenantID != ctxTenant {
		return autherrors.ErrInsufficientPermissions
	}

	return nil
}

// EnsureTenantHeader clones the given headers and sets the tenant id header when absent.
func EnsureTenantHeader(headers http.Header, tenantID string) http.Header {
	var h http.Header
	if headers == nil {
		h = http.Header{}
	} else {
		h = headers.Clone()
	}

	if tenantID != "" && h.Get(TenantIDHeader) == "" {
		h.Set(TenantIDHeader, tenantID)
	}

	return h
}
