package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

func TestTenantFromContextSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = req
	c.Set("claims", &types.Claims{TenantID: "tenant-123", Service: "billing-service"})

	tenantID, ok := tenantFromContext(c)
	if !ok {
		t.Fatalf("expected tenant to be resolved")
	}
	if tenantID != "tenant-123" {
		t.Fatalf("expected tenant-123, got %s", tenantID)
	}

	if got := c.Request.Header.Get(authmw.TenantIDHeader); got != "tenant-123" {
		t.Fatalf("expected tenant header to be set, got %s", got)
	}

	ctxTenant, err := authmw.TenantIDFromContext(c.Request.Context())
	if err != nil {
		t.Fatalf("expected tenant in request context: %v", err)
	}
	if ctxTenant != "tenant-123" {
		t.Fatalf("expected tenant in context to be tenant-123, got %s", ctxTenant)
	}
}

func TestTenantFromContextUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = req

	if tenantID, ok := tenantFromContext(c); ok || tenantID != "" {
		t.Fatalf("expected unauthorized flow")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}
