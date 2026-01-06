package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

// Verifica que o cliente inclui o X-Tenant-Id quando presente no contexto.
func TestEmbedSetsTenantHeader(t *testing.T) {
	var captured http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"embedding":[1.0,2.0]}]}`))
	}))
	t.Cleanup(srv.Close)

	client, err := New(Config{APIKey: "key", Model: "m", BaseURL: srv.URL, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("unexpected client error: %v", err)
	}

	ctx := authmw.WithTenantID(context.Background(), "tenant-123")
	if _, err := client.Embed(ctx, []string{"hello"}); err != nil {
		t.Fatalf("embed failed: %v", err)
	}

	if got := captured.Get(authmw.TenantIDHeader); got != "tenant-123" {
		t.Fatalf("expected tenant header, got %q", got)
	}
	if auth := captured.Get("Authorization"); auth == "" {
		t.Fatalf("expected Authorization header set")
	}
}

// Verifica que o cabeçalho de tenant não é enviado quando ausente no contexto.
func TestEmbedOmitsTenantHeaderWhenMissing(t *testing.T) {
	var captured http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"embedding":[1.0,2.0]}]}`))
	}))
	t.Cleanup(srv.Close)

	client, err := New(Config{APIKey: "key", Model: "m", BaseURL: srv.URL, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("unexpected client error: %v", err)
	}

	if _, err := client.Embed(context.Background(), []string{"hello"}); err != nil {
		t.Fatalf("embed failed: %v", err)
	}

	if got := captured.Get(authmw.TenantIDHeader); got != "" {
		t.Fatalf("expected no tenant header, got %q", got)
	}
}
