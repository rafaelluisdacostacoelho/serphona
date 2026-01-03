package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

// TestHTTPClientSetsTenantHeader verifies tenant header is set when available in context
func TestHTTPClientSetsTenantHeader(t *testing.T) {
	// Create a test server that captures the request
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Create HTTP client
	client := NewHTTPClient(10*time.Second, "tools-gateway", "tools-gateway-1", "")

	// Create tool configuration
	tool := &entity.Tool{
		ID:                uuid.New(),
		Name:              "Test Tool",
		BaseURL:           server.URL,
		EndpointPath:      "/api/test",
		Method:            entity.HTTPMethodGET,
		InputSchema:       json.RawMessage(`{}`),
		OutputSchema:      json.RawMessage(`{}`),
		AuthType:          entity.AuthTypeNone.String(),
		IsActive:          true,
		MaxRetries:        1,
		RetryDelaySeconds: 0,
	}

	// Create context with tenant ID
	ctx := middleware.WithTenantID(context.Background(), "tenant-123")

	// Execute request
	resp, err := client.Execute(ctx, tool, map[string]interface{}{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify tenant header was set
	tenantHeader := capturedHeaders.Get(middleware.TenantIDHeader)
	if tenantHeader != "tenant-123" {
		t.Errorf("expected X-Tenant-Id header 'tenant-123', got '%s'", tenantHeader)
	}
}

// TestHTTPClientNoTenantHeaderWhenMissing verifies no tenant header is set when context lacks tenant
func TestHTTPClientNoTenantHeaderWhenMissing(t *testing.T) {
	// Create a test server that captures the request
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Create HTTP client
	client := NewHTTPClient(10*time.Second, "tools-gateway", "tools-gateway-1", "")

	// Create tool configuration
	tool := &entity.Tool{
		ID:                uuid.New(),
		Name:              "Test Tool",
		BaseURL:           server.URL,
		EndpointPath:      "/api/test",
		Method:            entity.HTTPMethodGET,
		InputSchema:       json.RawMessage(`{}`),
		OutputSchema:      json.RawMessage(`{}`),
		AuthType:          entity.AuthTypeNone.String(),
		IsActive:          true,
		MaxRetries:        1,
		RetryDelaySeconds: 0,
	}

	// Execute request with context that has no tenant
	resp, err := client.Execute(context.Background(), tool, map[string]interface{}{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify tenant header was NOT set
	tenantHeader := capturedHeaders.Get(middleware.TenantIDHeader)
	if tenantHeader != "" {
		t.Errorf("expected no X-Tenant-Id header, got '%s'", tenantHeader)
	}
}

// TestHTTPClientContextTenantTakesPrecedence verifies context tenant overrides tool header
func TestHTTPClientContextTenantTakesPrecedence(t *testing.T) {
	// Create a test server that captures the request
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Create HTTP client
	client := NewHTTPClient(10*time.Second, "tools-gateway", "tools-gateway-1", "")

	// Create tool configuration with default header
	tool := &entity.Tool{
		ID:                uuid.New(),
		Name:              "Test Tool",
		BaseURL:           server.URL,
		EndpointPath:      "/api/test",
		Method:            entity.HTTPMethodGET,
		InputSchema:       json.RawMessage(`{}`),
		OutputSchema:      json.RawMessage(`{}`),
		AuthType:          entity.AuthTypeNone.String(),
		IsActive:          true,
		MaxRetries:        1,
		RetryDelaySeconds: 0,
		Headers: json.RawMessage(`{
			"X-Tenant-Id": "tenant-tool-config",
			"X-Custom-Header": "custom-value"
		}`),
	}

	// Create context with different tenant ID
	ctx := middleware.WithTenantID(context.Background(), "tenant-context")

	// Execute request
	resp, err := client.Execute(ctx, tool, map[string]interface{}{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify context tenant takes precedence over tool config header
	// This ensures proper tenant enforcement for cross-service calls
	tenantHeader := capturedHeaders.Get(middleware.TenantIDHeader)
	if tenantHeader != "tenant-context" {
		t.Errorf("expected X-Tenant-Id header 'tenant-context' (context takes precedence), got '%s'", tenantHeader)
	}
}

// TestHTTPClientSetsServiceIdentity verifies service identity headers are injected when configured
func TestHTTPClientSetsServiceIdentity(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(10*time.Second, "tools-gateway", "tools-gateway-1", "")
	tool := &entity.Tool{
		ID:                uuid.New(),
		Name:              "Test Tool",
		BaseURL:           server.URL,
		EndpointPath:      "/api/test",
		Method:            entity.HTTPMethodGET,
		InputSchema:       json.RawMessage(`{}`),
		OutputSchema:      json.RawMessage(`{}`),
		AuthType:          entity.AuthTypeNone.String(),
		IsActive:          true,
		MaxRetries:        1,
		RetryDelaySeconds: 0,
	}

	_, err := client.Execute(context.Background(), tool, map[string]interface{}{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := capturedHeaders.Get("X-Service-Name"); got != "tools-gateway" {
		t.Errorf("expected X-Service-Name 'tools-gateway', got '%s'", got)
	}
	if got := capturedHeaders.Get("X-Service-Instance"); got != "tools-gateway-1" {
		t.Errorf("expected X-Service-Instance 'tools-gateway-1', got '%s'", got)
	}
}
