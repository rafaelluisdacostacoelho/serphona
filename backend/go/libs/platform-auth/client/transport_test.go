package client

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequestIDRoundTripperGeneratesID(t *testing.T) {
	rt := instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get(requestIDHeader) == "" {
			t.Fatalf("request id header not set")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	}), nil, "")

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, _ = rt.RoundTrip(req)
}

func TestRequestIDRoundTripperPropagatesFromContext(t *testing.T) {
	rt := instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get(requestIDHeader); got != "req-ctx" {
			t.Fatalf("expected request id from context, got %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	}), nil, "")

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	req = req.WithContext(middleware.WithRequestID(context.Background(), "req-ctx"))
	_, _ = rt.RoundTrip(req)
}

func TestRequestIDRoundTripperPreservesExistingHeader(t *testing.T) {
	rt := instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get(requestIDHeader); got != "req-existing" {
			t.Fatalf("expected existing request id, got %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	}), nil, "")

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set(requestIDHeader, "req-existing")
	_, _ = rt.RoundTrip(req)
}

func TestServiceIdentityHeaders(t *testing.T) {
	headers := map[string]string{
		serviceNameHeader:     "auth-gateway",
		serviceInstanceHeader: "auth-gateway-1",
	}

	rt := instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get(serviceNameHeader); got != "auth-gateway" {
			t.Fatalf("expected service name header, got %q", got)
		}
		if got := r.Header.Get(serviceInstanceHeader); got != "auth-gateway-1" {
			t.Fatalf("expected service instance header, got %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	}), headers, "")

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, _ = rt.RoundTrip(req)
}

func TestTenantHeaderPropagatesFromContext(t *testing.T) {
	rt := instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get(middleware.TenantIDHeader); got != "t-123" {
			t.Fatalf("expected tenant header t-123, got %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	}), nil, "")

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	req = req.WithContext(middleware.WithTenantID(context.Background(), "t-123"))
	_, _ = rt.RoundTrip(req)
}

func TestTenantHeaderPreservesExisting(t *testing.T) {
	rt := instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get(middleware.TenantIDHeader); got != "t-existing" {
			t.Fatalf("expected existing tenant header preserved, got %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	}), nil, "")

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set(middleware.TenantIDHeader, "t-existing")
	req = req.WithContext(middleware.WithTenantID(context.Background(), "t-ctx"))
	_, _ = rt.RoundTrip(req)
}

func TestStaticBearerAppliedWhenMissing(t *testing.T) {
	rt := instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer service-token" {
			t.Fatalf("expected static bearer token, got %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	}), nil, "service-token")

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, _ = rt.RoundTrip(req)
}

func TestStaticBearerDoesNotOverrideExisting(t *testing.T) {
	rt := instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer user-token" {
			t.Fatalf("expected to preserve existing bearer, got %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	}), nil, "service-token")

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("Authorization", "Bearer user-token")
	_, _ = rt.RoundTrip(req)
}

func TestRetryRoundTripperRetriesOn5xx(t *testing.T) {
	attempts := 0
	rt := &retryRoundTripper{
		next: instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: http.NoBody}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		}), nil, ""),
		cfg: RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 2 * time.Millisecond},
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected final status ok, got %d", resp.StatusCode)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetryRoundTripperRetriesOn429(t *testing.T) {
	attempts := 0
	rt := &retryRoundTripper{
		next: instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return &http.Response{StatusCode: http.StatusTooManyRequests, Body: http.NoBody}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		}), nil, ""),
		cfg: RetryConfig{MaxAttempts: 2, BaseDelay: 1 * time.Millisecond, MaxDelay: 2 * time.Millisecond},
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected final status ok, got %d", resp.StatusCode)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts on 429, got %d", attempts)
	}
}

func TestRetryRoundTripperSkipsNonRetriable(t *testing.T) {
	attempts := 0
	rt := &retryRoundTripper{
		next: instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{StatusCode: http.StatusNotImplemented, Body: http.NoBody}, nil
		}), nil, ""),
		cfg: RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 2 * time.Millisecond},
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", resp.StatusCode)
	}
	if attempts != 1 {
		t.Fatalf("expected single attempt for non-retriable status, got %d", attempts)
	}
}

func TestRetryRoundTripperStopsAfterMaxAttempts(t *testing.T) {
	attempts := 0
	rt := &retryRoundTripper{
		next: instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.New("dial error")
		}), nil, ""),
		cfg: RetryConfig{MaxAttempts: 2, BaseDelay: 1 * time.Millisecond, MaxDelay: 2 * time.Millisecond},
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatalf("expected error after retries")
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestInstrumentTransportPropagatesTraceparent(t *testing.T) {
	var gotTraceparent string
	rt := instrumentTransport(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		gotTraceparent = r.Header.Get("Traceparent")
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	}), nil, "")

	tp := tracesdk.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	ctx, span := tp.Tracer("test").Start(context.Background(), "outbound")
	defer span.End()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	_, _ = rt.RoundTrip(req)

	if gotTraceparent == "" {
		t.Fatalf("expected traceparent header to be propagated")
	}
}

func TestWithDefaultTransportWrapsNil(t *testing.T) {
	rt := WithDefaultTransport(nil)
	if rt == nil {
		t.Fatalf("expected transport")
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err == nil || resp != nil {
		return // can't assert more without making a real call; ensure it doesn't panic
	}
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	cb := &circuitBreakerRoundTripper{
		next: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("dial error")
		}),
		cfg: CircuitBreakerConfig{FailureThreshold: 2, Cooldown: 50 * time.Millisecond},
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if _, err := cb.RoundTrip(req); err == nil {
		t.Fatalf("expected failure 1")
	}
	if _, err := cb.RoundTrip(req); err == nil {
		t.Fatalf("expected failure 2")
	}

	if _, err := cb.RoundTrip(req); err == nil {
		t.Fatalf("expected circuit open error")
	}

	time.Sleep(60 * time.Millisecond)
	cb.next = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})

	resp, err := cb.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected success after cooldown, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected ok after cooldown, got %d", resp.StatusCode)
	}
}
