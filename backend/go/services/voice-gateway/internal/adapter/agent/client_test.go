package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
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
	client := NewClient(server.URL, mustToken(t, "internal"), "voice-gateway", "vg-1", "internal", logger)

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

	// Verify service identity headers were set
	if got := capturedHeaders.Get("X-Service-Name"); got != "voice-gateway" {
		t.Errorf("expected X-Service-Name 'voice-gateway', got '%s'", got)
	}
	if got := capturedHeaders.Get("X-Service-Instance"); got != "vg-1" {
		t.Errorf("expected X-Service-Instance 'vg-1', got '%s'", got)
	}
}

func TestAgentClientCreateConversationUnexpectedStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, mustToken(t, "internal"), "voice-gateway", "vg-1", "internal", zap.NewNop())
	client.httpClient = server.Client()

	_, err := client.CreateConversation(context.Background(), uuid.New(), "agent-123")
	if err == nil || !strings.Contains(err.Error(), "unexpected status code") {
		t.Fatalf("expected unexpected status error, got %v", err)
	}
}

func TestAgentClientCreateConversationAudienceMismatch(t *testing.T) {
	t.Parallel()

	client := NewClient("http://example", mustToken(t, "other"), "voice-gateway", "vg-1", "internal", zap.NewNop())

	if _, err := client.CreateConversation(context.Background(), uuid.New(), "agent-123"); err == nil || !strings.Contains(err.Error(), "audience") {
		t.Fatalf("expected audience mismatch error, got %v", err)
	}
}

func TestAgentClientSubmitTurnSuccessAndError(t *testing.T) {
	t.Parallel()

	conversationID := uuid.New()
	turnID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"conversation_id":"` + conversationID.String() + `","turn_id":"` + turnID.String() + `","agent_response":"hi","state":"in_progress"}`))
		case http.MethodGet:
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, mustToken(t, "internal"), "voice-gateway", "vg-1", "internal", zap.NewNop())
	client.httpClient = server.Client()

	resp, err := client.SubmitTurn(context.Background(), conversationID, "hello", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AgentResponse != "hi" || resp.TurnID != turnID {
		t.Fatalf("unexpected turn response: %+v", resp)
	}

	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})

	if _, err := client.SubmitTurn(context.Background(), conversationID, "fail", nil); err == nil || !strings.Contains(err.Error(), "unexpected status code") {
		t.Fatalf("expected unexpected status error, got %v", err)
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
	client := NewClient(server.URL, mustToken(t, "internal"), "voice-gateway", "vg-1", "internal", logger)

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

	// Service identity should still be present
	if got := capturedHeaders.Get("X-Service-Name"); got != "voice-gateway" {
		t.Errorf("expected X-Service-Name 'voice-gateway', got '%s'", got)
	}
	if got := capturedHeaders.Get("X-Service-Instance"); got != "vg-1" {
		t.Errorf("expected X-Service-Instance 'vg-1', got '%s'", got)
	}
}

func mustToken(t *testing.T, aud string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Audience: []string{aud}})
	signed, err := tok.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}
