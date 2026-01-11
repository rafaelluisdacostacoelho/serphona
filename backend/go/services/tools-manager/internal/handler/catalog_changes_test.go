package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	authtypes "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
	"go.uber.org/zap"

	"tools-manager/internal/events"
)

// Verifies webhook payload emission and tenant filtering on /catalog/changes.
func TestNotifierWebhookAndChangesFilter(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	received := make(chan events.ChangeEvent, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var evt events.ChangeEvent
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			t.Errorf("decode webhook body: %v", err)
		} else {
			received <- evt
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := events.NewNotifier(srv.URL)

	now := time.Now().UTC().Truncate(time.Second)
	evtTenantA := events.ChangeEvent{Type: "tool.updated", TenantID: "tenant-a", ToolID: "tool-1", VersionID: "v1", DiffHash: "hash-a", At: now}
	evtTenantB := events.ChangeEvent{Type: "tool.updated", TenantID: "tenant-b", ToolID: "tool-2", VersionID: "v2", DiffHash: "hash-b", At: now}

	notifier.Notify(evtTenantA)
	notifier.Notify(evtTenantB)

	select {
	case got := <-received:
		if got.TenantID != evtTenantA.TenantID || got.ToolID != evtTenantA.ToolID || got.DiffHash != evtTenantA.DiffHash {
			t.Fatalf("webhook payload mismatch: got %+v want %+v", got, evtTenantA)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("webhook payload not received")
	}

	handler := NewCatalogHandler(zap.NewNop(), nil, notifier)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/catalog/changes", nil)
	c.Request = req
	c.Set("claims", &authtypes.Claims{TenantID: "tenant-a", Role: "admin"})

	handler.Changes(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from /catalog/changes, got %d", w.Code)
	}

	var resp struct {
		Data struct {
			Items []events.ChangeEvent `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode changes response: %v", err)
	}

	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected 1 event for tenant-a, got %d", len(resp.Data.Items))
	}
	if resp.Data.Items[0].TenantID != "tenant-a" || resp.Data.Items[0].ToolID != "tool-1" {
		t.Fatalf("unexpected event in response: %+v", resp.Data.Items[0])
	}
}
