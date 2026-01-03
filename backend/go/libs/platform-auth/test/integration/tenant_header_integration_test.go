//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	authclient "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

// TestTenantHeaderPropagation ensures the default transport injects tenant headers when present in context
// and respects existing tenant headers when already set.
func TestTenantHeaderPropagation(t *testing.T) {
	captured := make(chan http.Header, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := &http.Client{Transport: authclient.WithDefaultTransport(nil)}

	// Case 1: tenant in context -> header must be injected
	reqWithTenant, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	reqWithTenant = reqWithTenant.WithContext(middleware.WithTenantID(context.Background(), "tenant-integration"))

	resp, err := client.Do(reqWithTenant)
	if err != nil {
		t.Fatalf("do request with tenant: %v", err)
	}
	_ = resp.Body.Close()

	headers := <-captured
	if got := headers.Get(middleware.TenantIDHeader); got != "tenant-integration" {
		t.Fatalf("expected tenant header 'tenant-integration', got '%s'", got)
	}

	// Case 2: existing tenant header should be preserved even if context has tenant
	reqPreserve, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("build request preserve: %v", err)
	}
	reqPreserve.Header.Set(middleware.TenantIDHeader, "tenant-existing")
	reqPreserve = reqPreserve.WithContext(middleware.WithTenantID(context.Background(), "tenant-integration"))

	resp, err = client.Do(reqPreserve)
	if err != nil {
		t.Fatalf("do request preserve: %v", err)
	}
	_ = resp.Body.Close()

	headers = <-captured
	if got := headers.Get(middleware.TenantIDHeader); got != "tenant-existing" {
		t.Fatalf("expected existing tenant header to be preserved, got '%s'", got)
	}

	// Case 3: no tenant in context -> header should not be injected
	reqNoTenant, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("build request no tenant: %v", err)
	}

	resp, err = client.Do(reqNoTenant)
	if err != nil {
		t.Fatalf("do request no tenant: %v", err)
	}
	_ = resp.Body.Close()

	headers = <-captured
	if got := headers.Get(middleware.TenantIDHeader); got != "" {
		t.Fatalf("expected no tenant header when context is missing, got '%s'", got)
	}
}
