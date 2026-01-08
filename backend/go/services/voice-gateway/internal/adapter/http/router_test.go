package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestHealthHandler(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "healthy") {
		t.Fatalf("expected body to contain healthy, got %s", rec.Body.String())
	}
}

func TestLivenessHandler(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec := httptest.NewRecorder()

	livenessHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "alive") {
		t.Fatalf("expected body to contain alive, got %s", rec.Body.String())
	}
}

func TestReadinessHandler(t *testing.T) {
	t.Parallel()

	originalRedis := checkRedisConnection
	originalKafka := checkKafkaConnection
	originalAsterisk := checkAsteriskConnection
	t.Cleanup(func() {
		checkRedisConnection = originalRedis
		checkKafkaConnection = originalKafka
		checkAsteriskConnection = originalAsterisk
	})

	cases := []struct {
		name        string
		redisErr    error
		kafkaErr    error
		asteriskErr error
		wantStatus  int
		wantBody    string
	}{
		{name: "ready", wantStatus: http.StatusOK, wantBody: "ready"},
		{name: "redis down", redisErr: errors.New("redis down"), wantStatus: http.StatusServiceUnavailable, wantBody: "redis_unavailable"},
		{name: "kafka down", kafkaErr: errors.New("kafka down"), wantStatus: http.StatusServiceUnavailable, wantBody: "kafka_unavailable"},
		{name: "asterisk down", asteriskErr: errors.New("asterisk down"), wantStatus: http.StatusServiceUnavailable, wantBody: "asterisk_unavailable"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				checkRedisConnection = originalRedis
				checkKafkaConnection = originalKafka
				checkAsteriskConnection = originalAsterisk
			})

			checkRedisConnection = func() error { return tc.redisErr }
			checkKafkaConnection = func() error { return tc.kafkaErr }
			checkAsteriskConnection = func() error { return tc.asteriskErr }

			req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
			rec := httptest.NewRecorder()

			readinessHandler(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("expected body to contain %s, got %s", tc.wantBody, rec.Body.String())
			}
		})
	}
}

func TestCORSMiddlewareBlocksDisallowedOrigin(t *testing.T) {
	logger := zap.NewNop()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := corsMiddleware([]string{"https://app.serphona.com"}, logger)(mux)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed origin, got %d", rec.Code)
	}
}

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	logger := zap.NewNop()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := corsMiddleware([]string{"https://app.serphona.com"}, logger)(mux)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://app.serphona.com")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for allowed origin, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.serphona.com" {
		t.Fatalf("expected allow-origin header, got %s", got)
	}
}

func TestCORSMiddlewarePreflightChecksMethod(t *testing.T) {
	logger := zap.NewNop()
	mux := http.NewServeMux()

	h := corsMiddleware([]string{"https://app.serphona.com"}, logger)(mux)
	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	req.Header.Set("Origin", "https://app.serphona.com")
	req.Header.Set("Access-Control-Request-Method", "PATCH")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed preflight method, got %d", rec.Code)
	}
}
