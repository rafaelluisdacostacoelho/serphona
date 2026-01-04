package middleware

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func TestRedactHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer secret")
	headers.Set("Cookie", "session=abc")
	headers.Set("Proxy-Authorization", "Bearer proxy")
	headers.Set("X-Access-Token", "access")
	headers.Set("X-Custom", "ok")

	redacted := RedactHeaders(headers)

	if got := redacted.Get("Authorization"); got != "[REDACTED]" {
		t.Fatalf("expected authorization to be redacted, got %q", got)
	}
	if got := redacted.Get("Cookie"); got != "[REDACTED]" {
		t.Fatalf("expected cookie to be redacted, got %q", got)
	}
	if got := redacted.Get("Proxy-Authorization"); got != "[REDACTED]" {
		t.Fatalf("expected proxy authorization to be redacted, got %q", got)
	}
	if got := redacted.Get("X-Access-Token"); got != "[REDACTED]" {
		t.Fatalf("expected access token to be redacted, got %q", got)
	}
	if got := redacted.Get("X-Custom"); got != "ok" {
		t.Fatalf("expected custom header to be preserved, got %q", got)
	}

	// Ensure header keys remain canonicalized
	if _, exists := redacted[http.CanonicalHeaderKey("Authorization")]; !exists {
		t.Fatalf("expected header keys to remain canonicalized, found lowercase key")
	}
}

func TestSafeRequestFields(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/v1/resource?token=secret", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("Authorization", "Bearer secret")
	r.Header.Set("Proxy-Authorization", "Bearer proxy")
	r.Header.Set("X-Trace", "abc")
	r = r.WithContext(WithRequestID(r.Context(), "req-123"))

	fields := SafeRequestFields(r)

	if fields["method"] != http.MethodGet {
		t.Fatalf("unexpected method %v", fields["method"])
	}
	if fields["path"] != "/v1/resource" {
		t.Fatalf("unexpected path %v", fields["path"])
	}
	if fields["request_id"] != "req-123" {
		t.Fatalf("unexpected request_id %v", fields["request_id"])
	}

	headers, ok := fields["headers"].(http.Header)
	if !ok {
		t.Fatalf("headers field not of type http.Header")
	}
	if headers.Get("Authorization") != "[REDACTED]" {
		t.Fatalf("authorization not redacted in safe fields")
	}
	if headers.Get("Proxy-Authorization") != "[REDACTED]" {
		t.Fatalf("proxy authorization not redacted in safe fields")
	}
	if headers.Get("X-Trace") != "abc" {
		t.Fatalf("non-sensitive header should remain, got %q", headers.Get("X-Trace"))
	}
}

func TestSafeRequestKeyValues(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/v1/resource", nil)
	r = r.WithContext(WithSafeRequestFields(r.Context(), map[string]any{"request_id": "req-1", "method": "GET"}))

	kvs := SafeRequestKeyValues(r.Context())
	if len(kvs) == 0 {
		t.Fatalf("expected kvs to be populated")
	}
	foundID := false
	for i := 0; i < len(kvs); i += 2 {
		if kvs[i] == "request_id" && kvs[i+1] == "req-1" {
			foundID = true
		}
	}
	if !foundID {
		t.Fatalf("request_id not present in kvs: %v", kvs)
	}
}

func TestSafeGRPCRequestFields(t *testing.T) {
	md := metadata.Pairs("authorization", "Bearer secret", "x-trace", "abc", "proxy-authorization", "Bearer proxy")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = peer.NewContext(ctx, &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 50051}})
	ctx = WithRequestID(ctx, "req-grpc")

	fields := SafeGRPCRequestFields(ctx, "/svc.Method")

	if fields["method"] != "/svc.Method" {
		t.Fatalf("unexpected method %v", fields["method"])
	}
	if fields["request_id"] != "req-grpc" {
		t.Fatalf("unexpected request_id %v", fields["request_id"])
	}
	if fields["remote_addr"] != "127.0.0.1:50051" {
		t.Fatalf("unexpected remote_addr %v", fields["remote_addr"])
	}

	meta, ok := fields["metadata"].(metadata.MD)
	if !ok {
		t.Fatalf("metadata field not of type metadata.MD")
	}
	if got := meta.Get("authorization"); len(got) == 0 || got[0] != "[REDACTED]" {
		t.Fatalf("authorization not redacted, got %v", got)
	}
	if got := meta.Get("proxy-authorization"); len(got) == 0 || got[0] != "[REDACTED]" {
		t.Fatalf("proxy authorization not redacted, got %v", got)
	}
	if got := meta.Get("x-trace"); len(got) == 0 || got[0] != "abc" {
		t.Fatalf("x-trace not preserved, got %v", got)
	}
}
