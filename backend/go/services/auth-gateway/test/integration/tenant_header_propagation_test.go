//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/service/tenant"
)

// Verifies tenant service propagates X-Tenant-Id via platform-auth service transport.
func TestTenantServicePropagatesTenantHeader(t *testing.T) {
	var capturedTenant string
	createdID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenant = r.Header.Get(authmw.TenantIDHeader)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": createdID.String()})
	}))
	defer srv.Close()

	svc := tenant.NewService(srv.URL, "auth-gateway", "integration", "", "")

	ctx := authmw.WithTenantID(context.Background(), "tenant-integration")
	if _, err := svc.CreateTenant(ctx, "Acme Corp", "owner@example.com", "starter", "", ""); err != nil {
		t.Fatalf("CreateTenant failed: %v", err)
	}

	if capturedTenant != "tenant-integration" {
		t.Fatalf("tenant header not propagated, got %q", capturedTenant)
	}
}
