package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestLimitBodySizeHTTPRejectsLarge(t *testing.T) {
	h := LimitBodySizeHTTP(4)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("12345"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestLimitBodySizeHTTPSucceeds(t *testing.T) {
	h := LimitBodySizeHTTP(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("1234"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLimitBodySizeGin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LimitBodySizeGin(4))
	r.POST("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("12345"))
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestCORSMiddlewareHTTP(t *testing.T) {
	policy := CORSPolicy{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           10 * time.Minute,
	}

	nextCalled := false
	h := CORSMiddlewareHTTP(policy)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	preflight := httptest.NewRequest(http.MethodOptions, "/", nil)
	preflight.Header.Set("Origin", "https://app.example.com")
	preflightRecorder := httptest.NewRecorder()
	h.ServeHTTP(preflightRecorder, preflight)

	if preflightRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", preflightRecorder.Code)
	}
	if origin := preflightRecorder.Header().Get("Access-Control-Allow-Origin"); origin != "https://app.example.com" {
		t.Fatalf("unexpected allow origin: %s", origin)
	}
	if vary := preflightRecorder.Header().Get("Vary"); vary != "Origin" {
		t.Fatalf("unexpected vary: %s", vary)
	}
	if nextCalled {
		t.Fatalf("preflight should not call next handler")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getReq.Header.Set("Origin", "https://app.example.com")
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRec.Code)
	}
	if origin := getRec.Header().Get("Access-Control-Allow-Origin"); origin != "https://app.example.com" {
		t.Fatalf("unexpected allow origin on GET: %s", origin)
	}
	if !nextCalled {
		t.Fatalf("next handler should have been called for GET")
	}
}

func TestGinCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	policy := CORSPolicy{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           5 * time.Minute,
	}

	r := gin.New()
	r.Use(GinCORS(policy))
	r.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	preflight := httptest.NewRecorder()
	preflightReq := httptest.NewRequest(http.MethodOptions, "/", nil)
	preflightReq.Header.Set("Origin", "https://app.example.com")
	r.ServeHTTP(preflight, preflightReq)

	if preflight.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", preflight.Code)
	}
	if origin := preflight.Header().Get("Access-Control-Allow-Origin"); origin != "https://app.example.com" {
		t.Fatalf("unexpected allow origin: %s", origin)
	}

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getReq.Header.Set("Origin", "https://app.example.com")
	r.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRec.Code)
	}
	if origin := getRec.Header().Get("Access-Control-Allow-Origin"); origin != "https://app.example.com" {
		t.Fatalf("unexpected allow origin on GET: %s", origin)
	}
}

func TestGinRecoveryHandlesPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinRecovery())
	r.GET("/panic", func(c *gin.Context) { panic("boom") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestLimitBodySizeGinAllowsSmall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LimitBodySizeGin(10))
	r.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("1234"))

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
