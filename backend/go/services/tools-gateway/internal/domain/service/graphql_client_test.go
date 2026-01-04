package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

func TestGraphQLClientSetsTenantHeader(t *testing.T) {
	// Create a test server that captures the request
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": {"test": "success"}}`))
	}))
	defer server.Close()

	// Create context with tenant ID
	ctx := middleware.WithTenantID(context.Background(), "tenant-123")

	// Use the GraphQL client
	clientGraphQL := NewGraphQLClient(10 * time.Second)

	// Call ExecuteQuery
	integration := &entity.Integration{
		BaseURL: server.URL,
		GraphQLConfig: &entity.GraphQLConfig{
			Endpoint: "/graphql",
		},
	}
	variables := map[string]interface{}{}
	_, err := clientGraphQL.ExecuteQuery(ctx, integration, "query { test }", variables, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tenant header was set
	if capturedHeaders.Get(middleware.TenantIDHeader) != "tenant-123" {
		t.Errorf("expected X-Tenant-ID header 'tenant-123', got '%s'", capturedHeaders.Get(middleware.TenantIDHeader))
	}
}
