package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
