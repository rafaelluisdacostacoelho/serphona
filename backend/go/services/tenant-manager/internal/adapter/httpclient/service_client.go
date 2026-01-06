package httpclient

import (
	"errors"
	"net/http"
	"time"

	"tenant-manager/internal/config"

	"github.com/golang-jwt/jwt/v5"
	authclient "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

// NewServiceClient builds an HTTP client with service identity, retries, and circuit breaker.
func NewServiceClient(serviceCfg config.ServiceConfig, timeout time.Duration, serviceToken string) *http.Client {
	return newClient(serviceCfg, timeout, serviceToken, serviceCfg.Audience, nil)
}

func newClient(serviceCfg config.ServiceConfig, timeout time.Duration, serviceToken string, expectedAudience string, base http.RoundTripper) *http.Client {
	transport := authclient.WithServiceTransport(base, serviceCfg.Name, serviceCfg.Instance)
	rt := &serviceTokenRoundTripper{next: transport, token: serviceToken, audience: expectedAudience}

	return &http.Client{
		Timeout:   timeout,
		Transport: rt,
	}
}

type serviceTokenRoundTripper struct {
	next     http.RoundTripper
	token    string
	audience string
}

func (rt *serviceTokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return rt.next.RoundTrip(req)
	}

	clone := req.Clone(req.Context())

	// Validate token regardless of whether it's empty
	if err := validateAudience(rt.token, rt.audience); err != nil {
		return nil, err
	}

	// Ensure Authorization header is set if token is valid
	if rt.token != "" && clone.Header.Get("Authorization") == "" {
		clone.Header.Set("Authorization", "Bearer "+rt.token)
	}

	// Add tenant header propagation using platform-auth helpers
	if tenantID, err := authmw.TenantIDFromContext(req.Context()); err == nil && tenantID != "" {
		clone.Header = authmw.EnsureTenantHeader(clone.Header, tenantID)
	}

	return rt.next.RoundTrip(clone)
}

func validateAudience(token, expected string) error {
	if expected == "" {
		return nil
	}

	if token == "" {
		return errors.New("invalid token: token is empty")
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims := &jwt.RegisteredClaims{}

	if _, _, err := parser.ParseUnverified(token, claims); err != nil {
		return errors.New("invalid token: parsing failed")
	}

	for _, aud := range claims.Audience {
		if aud == expected {
			return nil
		}
	}
	return errors.New("invalid token: audience mismatch")
}
