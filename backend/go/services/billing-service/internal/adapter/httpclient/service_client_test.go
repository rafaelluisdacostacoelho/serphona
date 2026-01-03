package httpclient

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	authclient "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/config"
)

func TestServiceClientInjectsIdentityAndToken(t *testing.T) {
	cfg := config.ServiceConfig{Name: "billing-service", Instance: "billing-1", AuthToken: mustSignedToken(t, "internal"), Audience: "internal"}
	capture := &captureRoundTripper{}

	client := &http.Client{
		Timeout: time.Second,
		Transport: &serviceTokenRoundTripper{
			next:     authclient.WithServiceTransport(capture, cfg.Name, cfg.Instance),
			token:    cfg.AuthToken,
			audience: cfg.Audience,
		},
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	if _, err := client.Do(req); err != nil {
		t.Fatalf("do request: %v", err)
	}

	if got := capture.req.Header.Get("Authorization"); got != "Bearer "+cfg.AuthToken {
		t.Fatalf("expected auth token, got %q", got)
	}
	if got := capture.req.Header.Get("X-Service-Name"); got != cfg.Name {
		t.Fatalf("expected service name header, got %q", got)
	}
	if got := capture.req.Header.Get("X-Service-Instance"); got != cfg.Instance {
		t.Fatalf("expected service instance header, got %q", got)
	}
	if got := capture.req.Header.Get("X-Request-Id"); got == "" {
		t.Fatalf("expected request id to be injected")
	}
}

func TestServiceClientFailsOnAudienceMismatch(t *testing.T) {
	cfg := config.ServiceConfig{Name: "billing-service", Instance: "billing-1", AuthToken: mustSignedToken(t, "other"), Audience: "internal"}
	capture := &captureRoundTripper{}

	client := &http.Client{
		Timeout: time.Second,
		Transport: &serviceTokenRoundTripper{
			next:     authclient.WithServiceTransport(capture, cfg.Name, cfg.Instance),
			token:    cfg.AuthToken,
			audience: cfg.Audience,
		},
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	if _, err := client.Do(req); err == nil {
		t.Fatalf("expected audience mismatch error")
	}
	if capture.req != nil {
		t.Fatalf("request should not be sent on audience mismatch")
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
