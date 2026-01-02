package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	authtypes "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

func TestTenantRateLimitBlocksPerTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	metrics := NewTenantRateMetrics()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		claims := &authtypes.Claims{TenantID: "tenant-1", UserID: "user-1"}
		ctx := authmw.WithClaims(c.Request.Context(), claims)
		ctx = authmw.WithTenantID(ctx, claims.TenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(TenantRateLimit(1, metrics))
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	first := httptest.NewRecorder()
	router.ServeHTTP(first, req.Clone(req.Context()))
	if first.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", first.Code)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, req.Clone(req.Context()))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request to be rate limited, got %d", second.Code)
	}

	snap := metrics.Snapshot()
	tenantMetrics, ok := snap["tenant-1"]
	if !ok {
		t.Fatalf("expected metrics for tenant-1")
	}
	if tenantMetrics.Allowed != 1 {
		t.Fatalf("expected allowed count 1, got %d", tenantMetrics.Allowed)
	}
	if tenantMetrics.Blocked != 1 {
		t.Fatalf("expected blocked count 1, got %d", tenantMetrics.Blocked)
	}
}
