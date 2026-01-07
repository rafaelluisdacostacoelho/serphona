//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Ensures the hardened HTTP transport propagates request/trace/tenant IDs and service identity end-to-end.
func TestHTTPClientPropagation(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	var gotRequestID, gotTenantID, gotTraceparent, gotSvc, gotSvcInstance string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestID = r.Header.Get("X-Request-Id")
		gotTenantID = r.Header.Get(middleware.TenantIDHeader)
		gotTraceparent = r.Header.Get("Traceparent")
		gotSvc = r.Header.Get("X-Service-Name")
		gotSvcInstance = r.Header.Get("X-Service-Instance")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	httpClient := &http.Client{Transport: client.WithServiceTransport(nil, "platform-auth-test", "platform-auth-test-1")}

	ctx := middleware.WithRequestID(context.Background(), "req-integration")
	ctx = middleware.WithTenantID(ctx, "tenant-integration")
	ctx, span := tp.Tracer("integration").Start(ctx, "outbound-call")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("http call failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if gotRequestID != "req-integration" {
		t.Fatalf("request id not propagated, got %q", gotRequestID)
	}
	if gotTenantID != "tenant-integration" {
		t.Fatalf("tenant id not propagated, got %q", gotTenantID)
	}
	if gotTraceparent == "" {
		t.Fatalf("traceparent not propagated")
	}
	if gotSvc != "platform-auth-test" {
		t.Fatalf("service name header missing, got %q", gotSvc)
	}
	if gotSvcInstance != "platform-auth-test-1" {
		t.Fatalf("service instance header missing, got %q", gotSvcInstance)
	}
}
