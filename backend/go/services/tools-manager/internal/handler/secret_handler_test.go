package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	authtypes "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
	"go.uber.org/zap"

	"tools-manager/internal/secret"
)

func TestSecretAccessIsTenantIsolated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := secret.NewStore("1234567890123456", 0, secret.NewMemoryProvider())
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	handler := NewSecretHandler(zap.NewNop(), store)

	// Put secret under tenant A
	putBody := map[string]string{"id": "api-key", "value": "secret"}
	putJSON, _ := json.Marshal(putBody)
	wPut := httptest.NewRecorder()
	cPut, _ := gin.CreateTestContext(wPut)
	reqPut := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewReader(putJSON))
	cPut.Request = reqPut
	cPut.Set("claims", &authtypes.Claims{TenantID: "tenant-a", UserID: "user-a"})

	handler.Put(cPut)
	if wPut.Code != http.StatusCreated {
		t.Fatalf("expected 201 on put, got %d", wPut.Code)
	}

	// Get secret with different tenant should 404
	wGet := httptest.NewRecorder()
	cGet, _ := gin.CreateTestContext(wGet)
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/api-key", nil)
	cGet.Params = gin.Params{gin.Param{Key: "id", Value: "api-key"}}
	cGet.Request = reqGet
	cGet.Set("claims", &authtypes.Claims{TenantID: "tenant-b", UserID: "user-b"})

	handler.Get(cGet)
	if wGet.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-tenant secret fetch, got %d", wGet.Code)
	}
}
