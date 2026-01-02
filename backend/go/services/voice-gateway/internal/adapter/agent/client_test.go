package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.uber.org/zap"
)

// TestAgentClientSetsTenantHeader verifies agent client includes X-Tenant-Id header in requests
func TestAgentClientSetsTenantHeader(t *testing.T) {
	// Create a test server that captures the request
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"conversation_id": "550e8400-e29b-41d4-a716-446655440000",
			"agent_id": "agent-123",
			"agent_name": "Test Agent",
			"state": "active",
			"created_at": "2024-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create agent client
	client := NewClient(server.URL, "test-token", logger)

	// Create context with tenant ID
	ctx := middleware.WithTenantID(context.Background(), "tenant-456")

	tenantID := uuid.New()
	_, err := client.CreateConversation(ctx, tenantID, "agent-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tenant header was set
	tenantHeader := capturedHeaders.Get(middleware.TenantIDHeader)
	if tenantHeader != "tenant-456" {
		t.Errorf("expected X-Tenant-Id header 'tenant-456', got '%s'", tenantHeader)
	}
}

// TestAgentClientNoTenantHeaderWhenMissing verifies no tenant header when context lacks tenant
func TestAgentClientNoTenantHeaderWhenMissing(t *testing.T) {
	// Create a test server that captures the request
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"conversation_id": "550e8400-e29b-41d4-a716-446655440000",
			"agent_id": "agent-123",
			"agent_name": "Test Agent",
			"state": "active",
			"created_at": "2024-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create agent client
	client := NewClient(server.URL, "test-token", logger)

	// Create context without tenant
	ctx := context.Background()

	tenantID := uuid.New()
	_, err := client.CreateConversation(ctx, tenantID, "agent-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tenant header was NOT set
	tenantHeader := capturedHeaders.Get(middleware.TenantIDHeader)
	if tenantHeader != "" {
		t.Errorf("expected no X-Tenant-Id header, got '%s'", tenantHeader)
	}
}
