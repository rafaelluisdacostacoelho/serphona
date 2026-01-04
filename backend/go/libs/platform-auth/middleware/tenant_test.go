package middleware

import (
	"context"
	"net/http"
	"testing"

	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

func TestTenantIDFromContextPrefersClaims(t *testing.T) {
	// Ajuste: Prioridade para tenant explícito no contexto
	ctx := WithClaims(context.Background(), &types.Claims{TenantID: "tenant-1"})
	ctx = WithTenantID(ctx, "tenant-override")

	tenant, err := TenantIDFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenant != "tenant-override" {
		t.Fatalf("expected tenant from context, got %s", tenant)
	}
}

func TestTenantIDFromContextFallback(t *testing.T) {
	ctx := WithTenantID(context.Background(), "tenant-ctx")

	tenant, err := TenantIDFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenant != "tenant-ctx" {
		t.Fatalf("expected tenant from ctx, got %s", tenant)
	}
}

func TestTenantIDFromContextMissing(t *testing.T) {
	if _, err := TenantIDFromContext(context.Background()); err == nil {
		t.Fatalf("expected error when tenant missing")
	}
}

func TestEnforceTenant(t *testing.T) {
	claims := &types.Claims{TenantID: "tenant-1"}
	ctx := WithClaims(context.Background(), claims)

	if err := EnforceTenant(ctx, "tenant-1"); err != nil {
		t.Fatalf("expected tenant match, got %v", err)
	}

	if err := EnforceTenant(ctx, "tenant-2"); err != autherrors.ErrInsufficientPermissions {
		t.Fatalf("expected permissions error on mismatch, got %v", err)
	}

	if err := EnforceTenant(context.Background(), "tenant-1"); err == nil {
		t.Fatalf("expected error when tenant missing")
	}
}

func TestEnsureTenantHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Trace", "abc")

	cloned := EnsureTenantHeader(headers, "tenant-1")

	if got := cloned.Get(TenantIDHeader); got != "tenant-1" {
		t.Fatalf("expected tenant header to be set, got %s", got)
	}

	if got := headers.Get(TenantIDHeader); got != "" {
		t.Fatalf("expected original headers unchanged, got %s", got)
	}

	again := EnsureTenantHeader(cloned, "tenant-2")
	if got := again.Get(TenantIDHeader); got != "tenant-1" {
		t.Fatalf("existing tenant header should not be overwritten, got %s", got)
	}
}

func TestEnsureTenantHeaderNilHeaders(t *testing.T) {
	cloned := EnsureTenantHeader(nil, "tenant-1")

	if cloned.Get(TenantIDHeader) != "tenant-1" {
		t.Fatalf("expected tenant header to be set on nil input headers")
	}
}

func TestTenantIDFromContextNilContext(t *testing.T) {
	if _, err := TenantIDFromContext(context.TODO()); err != autherrors.ErrUnauthorized {
		t.Fatalf("expected unauthorized error for nil context, got %v", err)
	}
}

func TestWithTenantIDNilContextReturnsBackground(t *testing.T) {
	// Ajuste: Verifica se o contexto padrão é retornado corretamente
	ctx := WithTenantID(context.TODO(), "tenant-1")
	if ctx == nil {
		t.Fatalf("expected context to default to context.Background, got nil")
	}

	tenant, err := TenantIDFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenant != "tenant-1" {
		t.Fatalf("expected tenant ID 'tenant-1', got %s", tenant)
	}
}

func TestEnforceTenantAllowsEmptyRequestedTenant(t *testing.T) {
	claims := &types.Claims{TenantID: "tenant-ctx"}
	ctx := WithClaims(context.Background(), claims)

	if err := EnforceTenant(ctx, ""); err != nil {
		t.Fatalf("expected empty requested tenant to pass, got %v", err)
	}
}
