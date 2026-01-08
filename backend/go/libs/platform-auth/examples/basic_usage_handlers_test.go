//go:build examples
// +build examples

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

func makeGinContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func TestGetDataHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, w := makeGinContext(t)
	c.Set("claims", &types.Claims{UserID: "u1", TenantID: "t1"})

	getData(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if body["userId"] != "u1" || body["tenantId"] != "t1" {
		t.Fatalf("unexpected body: %v", body)
	}
}

func TestListUsersHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, w := makeGinContext(t)
	c.Set("claims", &types.Claims{Email: "admin@example.com"})

	listUsers(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetAdminReportsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, w := makeGinContext(t)

	getAdminReports(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetSystemInfoHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, w := makeGinContext(t)

	getSystemInfo(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// sanity check that claims helpers still work when middleware has placed claims
func TestGetDataWithClaimsHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, w := makeGinContext(t)
	// simulate middleware context enrichment
	c.Set("claims", &types.Claims{UserID: "user-123", TenantID: "tenant-456"})
	c.Request = c.Request.WithContext(middleware.WithClaims(c.Request.Context(), &types.Claims{UserID: "user-123", TenantID: "tenant-456"}))

	getData(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
