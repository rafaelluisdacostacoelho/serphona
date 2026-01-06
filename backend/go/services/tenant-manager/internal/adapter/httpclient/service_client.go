package httpclient

import (
	"errors"
	"fmt"
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

	clone := req.Clone(req.Context())

	// Debug: Log initial headers
	fmt.Printf("Initial headers: %+v\n", req.Header)

	// Validate token regardless of whether it's empty
	if err := validateAudience(rt.token, rt.audience); err != nil {
		fmt.Printf("Token validation error: %v\n", err)
		return nil, err
	}

	// Ensure Authorization header is set if token is valid
	if rt.token != "" && clone.Header.Get("Authorization") == "" {
		clone.Header.Set("Authorization", "Bearer "+rt.token)
		fmt.Printf("Authorization header set: %s\n", clone.Header.Get("Authorization"))
	} else {
		fmt.Printf("Authorization header already set or token is empty\n")
	}

	// Debug: Log headers after setting Authorization
	fmt.Printf("Headers after Authorization: %+v\n", clone.Header)

	// Add tenant header propagation
	// Debug: Log tenant_id from context
	tenantID, ok := req.Context().Value("tenant_id").(string)
	if ok && tenantID != "" {
		fmt.Printf("tenant_id from context: %v\n", tenantID)
		clone.Header.Set("X-Tenant-ID", tenantID)
	} else {
		fmt.Printf("tenant_id not found or invalid in context\n")
	}

	// Debug: Log final headers before sending
	fmt.Printf("Final headers: %+v\n", clone.Header)

	return rt.next.RoundTrip(clone)
}

func validateAudience(token, expected string) error {
	fmt.Printf("validateAudience called with token: '%s', expected: '%s'\n", token, expected)

	if expected == "" {
		fmt.Printf("No expected audience provided\n")
		return nil
	}

	if token == "" {
		fmt.Printf("Token is empty\n")
		return errors.New("invalid token: token is empty")
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims := &jwt.RegisteredClaims{}

	// Debug: Log token parsing
	fmt.Printf("Validating token: %s\n", token)

	if _, _, err := parser.ParseUnverified(token, claims); err != nil {
		fmt.Printf("Token parsing error: %v\n", err)
		return errors.New("invalid token: parsing failed")
	}

	fmt.Printf("Parsed claims: %+v\n", claims)

	for _, aud := range claims.Audience {
		if aud == expected {
			fmt.Printf("Audience matched: %s\n", aud)
			return nil
		}
	}

	fmt.Printf("Audience mismatch: expected %s, got %v\n", expected, claims.Audience)
	return errors.New("invalid token: audience mismatch")
}
