package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"tenant-manager/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

func TestServiceClientInjectsHeadersAndToken(t *testing.T) {
	serviceCfg := config.ServiceConfig{Name: "tenant-manager", Instance: "tm-1", Audience: "internal"}
	token := mustSignedToken(t, "internal")

	capture := &captureRoundTripper{}
	client := newClient(serviceCfg, time.Second, token, serviceCfg.Audience, capture)

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	if _, err := client.Do(req); err != nil {
		t.Fatalf("client do failed: %v", err)
	}

	if got := capture.req.Header.Get("Authorization"); got != "Bearer "+token {
		t.Fatalf("expected bearer token header, got %q", got)
	}
	if got := capture.req.Header.Get("X-Service-Name"); got != serviceCfg.Name {
		t.Fatalf("expected service name header, got %q", got)
	}
	if got := capture.req.Header.Get("X-Service-Instance"); got != serviceCfg.Instance {
		t.Fatalf("expected service instance header, got %q", got)
	}
	if got := capture.req.Header.Get("X-Request-Id"); got == "" {
		t.Fatalf("expected request id header to be set")
	}
}

func TestServiceClientFailsOnAudienceMismatch(t *testing.T) {
	serviceCfg := config.ServiceConfig{Name: "tenant-manager", Instance: "tm-1", Audience: "internal"}
	token := mustSignedToken(t, "other")

	capture := &captureRoundTripper{}
	client := newClient(serviceCfg, time.Second, token, serviceCfg.Audience, capture)

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	if _, err := client.Do(req); err == nil {
		t.Fatalf("expected audience mismatch error")
	}
	if capture.req != nil {
		t.Fatalf("request should not have been sent on audience mismatch")
	}
}

func TestServiceClientInjectsTenantHeader(t *testing.T) {
	serviceCfg := config.ServiceConfig{Name: "tenant-manager", Instance: "tm-1", Audience: "internal"}
	token := mustSignedToken(t, "internal")

	capture := &captureRoundTripper{}
	client := newClient(serviceCfg, time.Second, token, serviceCfg.Audience, capture)

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req = req.WithContext(context.WithValue(req.Context(), "tenant_id", "test-tenant"))

	if _, err := client.Do(req); err != nil {
		t.Fatalf("client do failed: %v", err)
	}

	if got := capture.req.Header.Get("X-Tenant-ID"); got != "test-tenant" {
		t.Fatalf("expected tenant header, got %q", got)
	}
}

func mustSignedToken(t *testing.T, aud string) string {
	t.Helper()

	claims := jwt.RegisteredClaims{
		Audience: jwt.ClaimStrings{aud},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return signedToken
}

func TestServiceClient_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		ctx           context.Context
		existingToken string
		expectHeaders map[string]string
		expectError   bool
	}{
		{
			name:          "with tenant in context",
			ctx:           context.WithValue(context.Background(), "tenant_id", "tenant-123"),
			existingToken: mustSignedToken(t, "expected-audience"),
			expectHeaders: map[string]string{
				"Authorization": "Bearer " + mustSignedToken(t, "expected-audience"),
				"X-Tenant-ID":   "tenant-123",
			},
			expectError: false,
		},
		{
			name:          "without tenant in context",
			ctx:           context.Background(),
			existingToken: mustSignedToken(t, "expected-audience"),
			expectHeaders: map[string]string{
				"Authorization": "Bearer " + mustSignedToken(t, "expected-audience"),
			},
			expectError: false,
		},
		{
			name:          "invalid token",
			ctx:           context.WithValue(context.Background(), "tenant_id", "tenant-123"),
			existingToken: "",
			expectHeaders: map[string]string{
				"X-Tenant-ID": "tenant-123",
			},
			expectError: true,
		},
		{
			name:          "expired token",
			ctx:           context.WithValue(context.Background(), "tenant_id", "tenant-456"),
			existingToken: mustSignedToken(t, "expired-audience"),
			expectHeaders: map[string]string{
				"Authorization": "Bearer " + mustSignedToken(t, "expired-audience"),
				"X-Tenant-ID":   "tenant-456",
			},
			expectError: true,
		},
		{
			name:          "custom headers present",
			ctx:           context.WithValue(context.Background(), "tenant_id", "tenant-789"),
			existingToken: mustSignedToken(t, "expected-audience"),
			expectHeaders: map[string]string{
				"Authorization":   "Bearer " + mustSignedToken(t, "expected-audience"),
				"X-Tenant-ID":     "tenant-789",
				"X-Custom-Header": "custom-value",
			},
			expectError: false,
		},
		{
			name:          "no tenant_id and no token",
			ctx:           context.Background(),
			existingToken: "",
			expectHeaders: map[string]string{},
			expectError:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fmt.Printf("Running test: %s\n", tt.name)
			fmt.Printf("Context: %+v\n", tt.ctx)
			fmt.Printf("Existing Token: %s\n", tt.existingToken)

			captureRT := &captureRoundTripper{}
			client := newClient(config.ServiceConfig{Name: "test-service", Instance: "test-instance"}, time.Second, tt.existingToken, "expected-audience", captureRT)
			req, _ := http.NewRequestWithContext(tt.ctx, http.MethodGet, "http://example.com", nil)

			// Add custom header to the request
			req.Header.Set("X-Custom-Header", "custom-value")

			_, err := client.Transport.RoundTrip(req)
			if (err != nil) != tt.expectError {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}

			// Skip header validation if an error occurred
			if err != nil {
				return
			}

			for key, expectedValue := range tt.expectHeaders {
				if captureRT.req.Header.Get(key) != expectedValue {
					t.Errorf("expected header %s: %s, got: %s", key, expectedValue, captureRT.req.Header.Get(key))
				}
			}
		})
	}
}

type captureRoundTripper struct {
	req *http.Request
}

func (rt *captureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.req = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     http.Header{},
	}, nil
}
