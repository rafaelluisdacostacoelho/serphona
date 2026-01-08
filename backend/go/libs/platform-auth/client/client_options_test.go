package client

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestWithRetryConfigNormalizes(t *testing.T) {
	cfg := RetryConfig{MaxAttempts: 0, BaseDelay: 0, MaxDelay: 0, Jitter: false}
	c := NewWithOptions("http://example", WithRetryConfig(cfg))

	if c.retryCfg.MaxAttempts != 1 {
		t.Fatalf("expected MaxAttempts normalized to 1, got %d", c.retryCfg.MaxAttempts)
	}
	if c.retryCfg.BaseDelay <= 0 {
		t.Fatalf("expected BaseDelay > 0")
	}
	if c.retryCfg.MaxDelay <= 0 {
		t.Fatalf("expected MaxDelay > 0")
	}
}

func TestWithStaticBearerTokenFromEnv(t *testing.T) {
	os.Setenv("STATIC_BEARER", "token-123")
	t.Cleanup(func() { os.Unsetenv("STATIC_BEARER") })

	c := NewWithOptions("http://example", WithStaticBearerTokenFromEnv("STATIC_BEARER"))

	if c.serviceBearer != "token-123" {
		t.Fatalf("expected serviceBearer from env, got %s", c.serviceBearer)
	}
}

func TestWithServiceIdentityOption(t *testing.T) {
	c := NewWithOptions("http://example", WithServiceIdentity("svc", "inst"))

	if c.serviceHeaders[serviceNameHeader] != "svc" {
		t.Fatalf("missing service name header")
	}
	if c.serviceHeaders[serviceInstanceHeader] != "inst" {
		t.Fatalf("missing service instance header")
	}
}

func TestWithServiceBearerFromEnv(t *testing.T) {
	os.Setenv("SVC_BEARER", "svc-token")
	t.Cleanup(func() { os.Unsetenv("SVC_BEARER") })

	c := New("http://example")
	c.WithServiceBearerFromEnv("SVC_BEARER")

	if c.serviceBearer != "svc-token" {
		t.Fatalf("expected bearer from env, got %s", c.serviceBearer)
	}

	c.WithServiceBearerFromEnv("MISSING_ENV")
	if c.serviceBearer != "svc-token" {
		t.Fatalf("unexpected change when env missing")
	}

	// empty env var should be a no-op
	c.WithServiceBearer("keep")
	c.WithServiceBearerFromEnv("")
	if c.serviceBearer != "keep" {
		t.Fatalf("expected service bearer unchanged when env var empty")
	}
}

type recordingRoundTripper struct {
	lastReq *http.Request
}

func (rt *recordingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.lastReq = req
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString("ok")), Header: http.Header{}}, nil
}

func TestWithServiceTransportInjectsHeaders(t *testing.T) {
	base := &recordingRoundTripper{}
	rt := WithServiceTransport(base, "svc", "inst")

	req, _ := http.NewRequest(http.MethodGet, "http://example", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("round trip error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}

	if base.lastReq == nil {
		t.Fatalf("expected base transport to receive request")
	}
	headers := base.lastReq.Header
	if headers.Get(requestIDHeader) == "" {
		t.Fatalf("expected request id header")
	}
	if headers.Get(serviceNameHeader) != "svc" {
		t.Fatalf("expected service name header")
	}
	if headers.Get(serviceInstanceHeader) != "inst" {
		t.Fatalf("expected service instance header")
	}
}

func TestWithDefaultTransportAddsRequestID(t *testing.T) {
	base := &recordingRoundTripper{}
	rt := WithDefaultTransport(base)

	req, _ := http.NewRequest(http.MethodGet, "http://example", nil)
	_, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("round trip error: %v", err)
	}
	if base.lastReq.Header.Get(requestIDHeader) == "" {
		t.Fatalf("expected request id header on wrapped transport")
	}
}

func TestWithCircuitBreakerConfigNormalizes(t *testing.T) {
	c := NewWithOptions("http://example", WithCircuitBreakerConfig(CircuitBreakerConfig{FailureThreshold: 0, Cooldown: 0}))

	if c.cbCfg.FailureThreshold != 5 {
		t.Fatalf("expected default failure threshold 5, got %d", c.cbCfg.FailureThreshold)
	}
	if c.cbCfg.Cooldown != 5*time.Second {
		t.Fatalf("expected default cooldown 5s, got %v", c.cbCfg.Cooldown)
	}
}

func TestWithRetryRebuildsClient(t *testing.T) {
	c := New("http://example")
	original := c.httpClient

	c.WithRetry(RetryConfig{MaxAttempts: 4})

	if c.retryCfg.MaxAttempts != 4 {
		t.Fatalf("expected retry attempts 4, got %d", c.retryCfg.MaxAttempts)
	}
	if c.httpClient == original {
		t.Fatalf("expected http client to rebuild")
	}
}

func TestWithCircuitBreakerRebuildsClient(t *testing.T) {
	c := New("http://example")
	original := c.httpClient

	c.WithCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 2, Cooldown: time.Second})

	if c.cbCfg.FailureThreshold != 2 {
		t.Fatalf("expected failure threshold 2, got %d", c.cbCfg.FailureThreshold)
	}
	if c.httpClient == original {
		t.Fatalf("expected http client to rebuild")
	}
}

func TestWithTLSRebuildsClient(t *testing.T) {
	c := New("http://example")
	original := c.httpClient

	tlsCfg := &tls.Config{ServerName: "example.com"}
	c.WithTLS(tlsCfg)

	if c.tlsConfig != tlsCfg {
		t.Fatalf("expected tls config to be set")
	}
	if c.httpClient == original {
		t.Fatalf("expected http client to rebuild")
	}
}

func TestWithServiceRebuildsClient(t *testing.T) {
	c := New("http://example")
	original := c.httpClient

	c.WithService("svc", "inst")

	if c.serviceHeaders[serviceNameHeader] != "svc" || c.serviceHeaders[serviceInstanceHeader] != "inst" {
		t.Fatalf("service headers not set")
	}
	if c.httpClient == original {
		t.Fatalf("expected http client to rebuild")
	}
}

func TestHTTPErrorString(t *testing.T) {
	err := &HTTPError{StatusCode: 418}
	if got := err.Error(); got != "unexpected status code: 418" {
		t.Fatalf("unexpected error string: %s", got)
	}
}

func TestReadErrorBodyNilSafe(t *testing.T) {
	if readErrorBody(nil) != "" {
		t.Fatalf("expected empty string for nil resp")
	}
	resp := &http.Response{}
	if readErrorBody(resp) != "" {
		t.Fatalf("expected empty string for nil body")
	}
}

func TestStatusCodeNilSafe(t *testing.T) {
	if statusCode(nil) != 0 {
		t.Fatalf("expected zero for nil response")
	}
}

func TestWithHTTPClientWrapsTransport(t *testing.T) {
	base := &recordingRoundTripper{}
	client := &http.Client{Transport: base}

	c := New("http://example")
	c.WithServiceBearer("svc-token")
	c.WithHTTPClient(client)

	req, _ := http.NewRequest(http.MethodGet, "http://example", nil)
	_, err := c.httpClient.Do(req)
	if err != nil {
		t.Fatalf("do request error: %v", err)
	}
	if base.lastReq == nil {
		t.Fatalf("expected wrapped transport to execute request")
	}
	if base.lastReq.Header.Get("Authorization") != "Bearer svc-token" {
		t.Fatalf("expected bearer token injected, got %s", base.lastReq.Header.Get("Authorization"))
	}
}

func TestWithTLSFilesMissingCA(t *testing.T) {
	c := New("http://example")
	_, err := c.WithTLSFiles("/nope/ca.pem", "", "")
	if err == nil {
		t.Fatalf("expected error for missing CA file")
	}
}
