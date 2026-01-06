package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/service"
)

// Tests ensure that outbound calls include the tenant header when a tenant is provided.

func TestExecuteToolSetsTenantHeader(t *testing.T) {
	tenantID := uuid.New()

	var captured http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"execution_id":"1","tool_name":"demo","status":"success"}`))
	}))
	t.Cleanup(srv.Close)

	client := NewToolsClient(srv.URL, "", "", "agent-orchestrator", "ao-1")

	_, err := client.ExecuteTool(context.Background(), &service.ToolExecutionRequest{
		ToolName:   "demo",
		Parameters: map[string]interface{}{"foo": "bar"},
		TenantID:   tenantID,
		UserID:     "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteTool returned error: %v", err)
	}

	if got := captured.Get(authmw.TenantIDHeader); got != tenantID.String() {
		t.Fatalf("expected tenant header %q, got %q", tenantID.String(), got)
	}
}

func TestGetAvailableToolsSetsTenantHeader(t *testing.T) {
	tenantID := uuid.New()

	var captured http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tools":[{"name":"demo"}]}`))
	}))
	t.Cleanup(srv.Close)

	client := NewToolsClient(srv.URL, "", "", "agent-orchestrator", "ao-1")

	if _, err := client.GetAvailableTools(context.Background(), tenantID); err != nil {
		t.Fatalf("GetAvailableTools returned error: %v", err)
	}

	if got := captured.Get(authmw.TenantIDHeader); got != tenantID.String() {
		t.Fatalf("expected tenant header %q, got %q", tenantID.String(), got)
	}
}

func TestValidateToolSetsTenantHeader(t *testing.T) {
	tenantID := uuid.New()

	var captured http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"valid":true}`))
	}))
	t.Cleanup(srv.Close)

	client := NewToolsClient(srv.URL, "", "", "agent-orchestrator", "ao-1")

	ok, err := client.ValidateTool(context.Background(), tenantID, "demo")
	if err != nil {
		t.Fatalf("ValidateTool returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected tool to be valid")
	}

	if got := captured.Get(authmw.TenantIDHeader); got != tenantID.String() {
		t.Fatalf("expected tenant header %q, got %q", tenantID.String(), got)
	}
}
