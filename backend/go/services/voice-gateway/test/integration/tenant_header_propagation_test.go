//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/tenant"
	"go.uber.org/zap"
)

// Verifies tenant manager client propagates X-Tenant-Id via platform-auth transport.
func TestTenantClientPropagatesTenantHeader(t *testing.T) {
	var capturedTenant string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenant = r.Header.Get(authmw.TenantIDHeader)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"did":"+15551234567","tenant_id":"00000000-0000-0000-0000-000000000123","enabled":true}`)
	}))
	defer srv.Close()

	client := tenant.NewClient(srv.URL, "", "voice-gateway", "voice-gateway-1", "", zap.NewNop())

	ctx := authmw.WithTenantID(context.Background(), "tenant-integration")
	if _, err := client.LookupDID(ctx, "+15551234567"); err != nil {
		t.Fatalf("lookup did failed: %v", err)
	}

	if capturedTenant != "tenant-integration" {
		t.Fatalf("tenant header not propagated, got %q", capturedTenant)
	}
}
