package httpclient

import (
	"errors"
	"net/http"
	"time"

	"tenant-manager/internal/config"

	"github.com/golang-jwt/jwt/v5"
	authclient "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"
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

	clone := req
	if rt.token != "" && req.Header.Get("Authorization") == "" {
		clone = req.Clone(req.Context())
		if clone.Header == nil {
			clone.Header = http.Header{}
		}
		if err := validateAudience(rt.token, rt.audience); err != nil {
			return nil, err
		}
		clone.Header.Set("Authorization", "Bearer "+rt.token)
	}

	return rt.next.RoundTrip(clone)
}

func validateAudience(token, expected string) error {
	if expected == "" {
		return nil
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims := &jwt.RegisteredClaims{}

	if _, _, err := parser.ParseUnverified(token, claims); err != nil {
		return err
	}

	for _, aud := range claims.Audience {
		if aud == expected {
			return nil
		}
	}

	return errors.New("service token audience mismatch")
}
