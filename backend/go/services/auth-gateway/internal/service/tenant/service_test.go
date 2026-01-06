package tenant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

// recordingRoundTripper captura headers das requisições para verificação em testes.
type recordingRoundTripper struct {
	base   http.RoundTripper
	header http.Header
}

func (r *recordingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r.header = req.Header.Clone()
	return r.base.RoundTrip(req)
}

func TestCreateTenantPropagaTenantHeader(t *testing.T) {
	tenantID := "tenant-123"
	createdID := uuid.New()

	// Servidor de teste simula o tenant-manager.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": createdID.String()})
	}))
	defer srv.Close()

	svc := NewService(srv.URL, "auth-gateway", "test", "", "")

	recorder := &recordingRoundTripper{base: srv.Client().Transport}
	svc.httpClient = &http.Client{Transport: recorder}

	ctx := authmw.WithTenantID(context.Background(), tenantID)
	gotID, err := svc.CreateTenant(ctx, "Acme", "user@example.com", "starter", "bill@example.com", "123")
	if err != nil {
		t.Fatalf("CreateTenant falhou: %v", err)
	}
	if gotID != createdID {
		t.Fatalf("ID retornado inesperado: got %s want %s", gotID, createdID)
	}

	if recorder.header.Get(authmw.TenantIDHeader) != tenantID {
		t.Fatalf("tenant header não propagado: got %q", recorder.header.Get(authmw.TenantIDHeader))
	}
}
