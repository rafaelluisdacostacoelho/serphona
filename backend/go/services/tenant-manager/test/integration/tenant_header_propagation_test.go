//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tenant-manager/internal/adapter/httpclient"
	"tenant-manager/internal/config"

	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

// Verifies service HTTP client propagates X-Tenant-Id when present in context.
func TestServiceClientPropagatesTenantHeader(t *testing.T) {
	var capturedTenant string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenant = r.Header.Get(authmw.TenantIDHeader)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.ServiceConfig{Name: "tenant-manager", Instance: "tenant-manager-1", Audience: ""}
	httpClient := httpclient.NewServiceClient(cfg, 2*time.Second, "")

	ctx := authmw.WithTenantID(context.Background(), "tenant-integration")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("http call failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if capturedTenant != "tenant-integration" {
		t.Fatalf("tenant header not propagated, got %q", capturedTenant)
	}
}
