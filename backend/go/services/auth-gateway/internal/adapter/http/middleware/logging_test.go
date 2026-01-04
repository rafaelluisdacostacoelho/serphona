package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRequestLoggerRedactsSensitiveHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	router := gin.New()
	router.Use(Correlation())
	router.Use(RequestLogger(logger))
	router.GET("/foo", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Cookie", "session=abc")
	req.Header.Set("X-Request-ID", "req-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	ctx := entries[0].ContextMap()

	headersVal, ok := ctx["headers"].(http.Header)
	if !ok {
		t.Fatalf("expected headers field present, got %T", ctx["headers"])
	}

	for _, key := range []string{"Authorization", "Cookie"} {
		vals := headersVal.Values(key)
		if len(vals) != 1 || vals[0] != "[REDACTED]" {
			t.Fatalf("expected %s to be redacted, got %v", key, vals)
		}
	}

	if reqID, ok := ctx["request_id"].(string); !ok || reqID != "req-123" {
		t.Fatalf("expected request_id req-123, got %v", ctx["request_id"])
	}
}

func TestCSRFProtectionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CSRFProtectionMiddleware())
	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Test missing CSRF token
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	// Test invalid CSRF token
	req = httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("X-CSRF-Token", "invalid-token")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	// Test valid CSRF token
	req = httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("X-CSRF-Token", "expected-csrf-token")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
