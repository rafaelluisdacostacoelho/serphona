package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	requestIDHeader       = "X-Request-Id"
	serviceNameHeader     = "X-Service-Name"
	serviceInstanceHeader = "X-Service-Instance"
)

// RetryConfig configures client-side retries with exponential backoff.
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Jitter      bool
}

// CircuitBreakerConfig controls failure-based short-circuiting of outbound calls.
type CircuitBreakerConfig struct {
	FailureThreshold int
	Cooldown         time.Duration
}

func defaultRetryConfig() RetryConfig {
	return RetryConfig{MaxAttempts: 3, BaseDelay: 100 * time.Millisecond, MaxDelay: 2 * time.Second, Jitter: true}
}

func (cfg RetryConfig) normalize() RetryConfig {
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 100 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 2 * time.Second
	}
	return cfg
}

func (cfg RetryConfig) backoff(attempt int) time.Duration {
	cfg = cfg.normalize()
	delay := cfg.BaseDelay << (attempt - 1)
	if delay > cfg.MaxDelay {
		delay = cfg.MaxDelay
	}
	if cfg.Jitter {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		return time.Duration(r.Int63n(int64(delay)))
	}
	return delay
}

// instrumentTransport wraps the provided RoundTripper to propagate request IDs, service identity, and trace context.
func instrumentTransport(base http.RoundTripper, serviceHeaders map[string]string, bearerToken string) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}

	return otelhttp.NewTransport(&requestIDRoundTripper{next: base, serviceHeaders: serviceHeaders, bearerToken: bearerToken})
}

type requestIDRoundTripper struct {
	next           http.RoundTripper
	serviceHeaders map[string]string
	bearerToken    string
}

func (rt *requestIDRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return rt.next.RoundTrip(req)
	}

	// Avoid mutating caller request.
	clone := req.Clone(req.Context())
	if clone.Header == nil {
		clone.Header = http.Header{}
	}

	if clone.Header.Get(requestIDHeader) == "" {
		clone.Header.Set(requestIDHeader, deriveRequestID(clone.Context()))
	}

	if tenantID, err := middleware.TenantIDFromContext(clone.Context()); err == nil && tenantID != "" {
		if clone.Header.Get(middleware.TenantIDHeader) == "" {
			clone.Header.Set(middleware.TenantIDHeader, tenantID)
		}
	}

	for k, v := range rt.serviceHeaders {
		if clone.Header.Get(k) == "" {
			clone.Header.Set(k, v)
		}
	}

	if rt.bearerToken != "" && clone.Header.Get("Authorization") == "" {
		clone.Header.Set("Authorization", "Bearer "+rt.bearerToken)
	}

	return rt.next.RoundTrip(clone)
}

// retryRoundTripper applies best-effort retries with exponential backoff for transient failures.
type retryRoundTripper struct {
	next http.RoundTripper
	cfg  RetryConfig
}

func (rt *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return rt.next.RoundTrip(req)
	}

	cfg := rt.cfg.normalize()
	bodyBytes, err := drainBody(req)
	if err != nil {
		return nil, err
	}

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		clone, cloneErr := cloneRequestWithBody(req, bodyBytes)
		if cloneErr != nil {
			return nil, cloneErr
		}

		resp, err := rt.next.RoundTrip(clone)
		if err != nil {
			if attempt == cfg.MaxAttempts {
				return nil, err
			}
			time.Sleep(cfg.backoff(attempt))
			continue
		}

		if !shouldRetry(resp.StatusCode) || attempt == cfg.MaxAttempts {
			return resp, nil
		}

		resp.Body.Close()
		time.Sleep(cfg.backoff(attempt))
	}

	return nil, nil
}

func shouldRetry(status int) bool {
	if status == 0 {
		return true
	}
	if status == http.StatusTooManyRequests {
		return true
	}
	return status >= 500 && status != http.StatusNotImplemented
}

type circuitBreakerRoundTripper struct {
	next         http.RoundTripper
	failureCount int
	stateOpen    bool
	openedAt     time.Time
	cfg          CircuitBreakerConfig
}

func (rt *circuitBreakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cfg := rt.cfg
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = 5 * time.Second
	}

	if rt.stateOpen {
		if time.Since(rt.openedAt) < cfg.Cooldown {
			return nil, fmt.Errorf("circuit breaker open")
		}
		rt.stateOpen = false
		rt.failureCount = 0
	}

	resp, err := rt.next.RoundTrip(req)
	if err != nil || shouldRetry(statusCode(resp)) {
		rt.failureCount++
		if rt.failureCount >= cfg.FailureThreshold {
			rt.stateOpen = true
			rt.openedAt = time.Now()
		}
		return resp, err
	}

	rt.failureCount = 0
	return resp, err
}

func statusCode(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

func drainBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	if err := req.Body.Close(); err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(data))
	return data, nil
}

func cloneRequestWithBody(req *http.Request, body []byte) (*http.Request, error) {
	clone := req.Clone(req.Context())
	if body != nil {
		clone.Body = io.NopCloser(bytes.NewReader(body))
		clone.ContentLength = int64(len(body))
		clone.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
	}
	return clone, nil
}

func wrapTransport(base http.RoundTripper, retryCfg RetryConfig, cbCfg CircuitBreakerConfig, serviceHeaders map[string]string, bearerToken string) http.RoundTripper {
	instrumented := instrumentTransport(base, serviceHeaders, bearerToken)
	cb := &circuitBreakerRoundTripper{next: instrumented, cfg: cbCfg}
	return &retryRoundTripper{
		next: cb,
		cfg:  retryCfg,
	}
}

func defaultTransport(tlsCfg *tls.Config) http.RoundTripper {
	if tlsCfg == nil {
		return http.DefaultTransport
	}
	if dt, ok := http.DefaultTransport.(*http.Transport); ok {
		clone := dt.Clone()
		clone.TLSClientConfig = tlsCfg
		return clone
	}
	return &http.Transport{TLSClientConfig: tlsCfg}
}

func deriveRequestID(ctx context.Context) string {
	if ctx != nil {
		if id, err := middleware.RequestIDFromContext(ctx); err == nil && id != "" {
			return id
		}
	}
	return uuid.NewString()
}

// WithServiceTransport returns a hardened transport (retry + circuit breaker + propagation)
// and injects service identity headers when provided.
func WithServiceTransport(base http.RoundTripper, serviceName, serviceInstance string) http.RoundTripper {
	serviceHeaders := map[string]string{}
	if serviceName != "" {
		serviceHeaders[serviceNameHeader] = serviceName
	}
	if serviceInstance != "" {
		serviceHeaders[serviceInstanceHeader] = serviceInstance
	}

	return wrapTransport(base, defaultRetryConfig(), CircuitBreakerConfig{FailureThreshold: 5, Cooldown: 5 * time.Second}, serviceHeaders, "")
}

// WithDefaultTransport returns a hardened transport (retry + circuit breaker + propagation) with defaults.
// Useful for callers that want instrumentation without constructing a full Client.
func WithDefaultTransport(base http.RoundTripper) http.RoundTripper {
	return WithServiceTransport(base, "", "")
}
