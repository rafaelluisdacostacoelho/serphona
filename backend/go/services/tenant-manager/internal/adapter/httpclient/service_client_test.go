package httpclient

import (
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

func mustSignedToken(t *testing.T, aud string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Audience: []string{aud}})
	signed, err := tok.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}
