package asterisk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestARIClientHTTPHealthCheckSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ari/asterisk/info" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "user" || pass != "pass" {
			t.Fatalf("unexpected basic auth: %s/%s", user, pass)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewARIClientHTTP(ARIConfig{URL: server.URL, Username: "user", Password: "pass", AppName: "app"}, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	client.httpClient = server.Client()

	if err := client.HealthCheck(context.Background()); err != nil {
		t.Fatalf("expected health check to pass, got %v", err)
	}
}

func TestARIClientHTTPHealthCheckFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("ari down"))
	}))
	defer server.Close()

	client, err := NewARIClientHTTP(ARIConfig{URL: server.URL, Username: "user", Password: "pass", AppName: "app"}, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	client.httpClient = server.Client()

	if err := client.HealthCheck(context.Background()); err == nil {
		t.Fatalf("expected health check to fail")
	}
}
