package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

func TestOAuth2ServiceSetsTenantHeader(t *testing.T) {
	// Create a test server that captures the request
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token": "test-token", "token_type": "bearer"}`))
	}))
	defer server.Close()

	// Create context with tenant ID
	ctx := middleware.WithTenantID(context.Background(), "tenant-123")

	// Create OAuth2 service
	oauth2Service := NewOAuth2Service(10 * time.Second)

	// Create integration configuration
	integration := &entity.Integration{
		OAuth2Config: &entity.OAuth2Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			TokenURL:     server.URL,
		},
	}

	// Call GetClientCredentialsToken
	oauth2Service.GetClientCredentialsToken(ctx, integration, uuid.New(), nil)

	// Verify tenant header was set
	if capturedHeaders.Get(middleware.TenantIDHeader) != "tenant-123" {
		t.Errorf("expected X-Tenant-ID header 'tenant-123', got '%s'", capturedHeaders.Get(middleware.TenantIDHeader))
	}
}
