package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

const (
	// EnvAuthGatewayURL is the default environment variable name used to configure the gateway URL.
	EnvAuthGatewayURL = "AUTH_GATEWAY_URL"
	// EnvServiceAuthToken is the environment variable used to inject a static bearer for internal calls.
	EnvServiceAuthToken = "SERVICE_AUTH_TOKEN"
	// EnvServiceName is the environment variable used to set the calling service name header.
	EnvServiceName = "SERVICE_NAME"
	// EnvServiceInstance is the environment variable used to set the calling service instance header.
	EnvServiceInstance = "SERVICE_INSTANCE"
)

// Client is an HTTP client used to communicate with auth-gateway.
type Client struct {
	baseURL          string
	httpClient       *http.Client
	customHTTPClient bool
	retryCfg         RetryConfig
	cbCfg            CircuitBreakerConfig
	tlsConfig        *tls.Config
	serviceHeaders   map[string]string
	serviceBearer    string
}

// New creates a new auth-gateway HTTP client.
func New(baseURL string) *Client {
	return NewWithOptions(baseURL)
}

// NewFromEnv creates a new client using EnvAuthGatewayURL and returns an error if missing.
func NewFromEnv() (*Client, error) {
	baseURL := os.Getenv(EnvAuthGatewayURL)
	if baseURL == "" {
		return nil, fmt.Errorf("environment variable %s not set", EnvAuthGatewayURL)
	}
	return New(baseURL), nil
}

// NewServiceClientFromEnv builds a client preloaded with service identity and optional static bearer for internal calls.
// If serviceName or serviceInstance are empty, the function falls back to ENV vars when present.
func NewServiceClientFromEnv(serviceName, serviceInstance string, opts ...Option) (*Client, error) {
	baseURL := os.Getenv(EnvAuthGatewayURL)
	if baseURL == "" {
		return nil, fmt.Errorf("environment variable %s not set", EnvAuthGatewayURL)
	}

	if serviceName == "" {
		serviceName = os.Getenv(EnvServiceName)
	}
	if serviceInstance == "" {
		serviceInstance = os.Getenv(EnvServiceInstance)
	}

	options := []Option{WithServiceIdentity(serviceName, serviceInstance)}
	if token := os.Getenv(EnvServiceAuthToken); token != "" {
		options = append(options, WithStaticBearerToken(token))
	}
	options = append(options, opts...)

	return NewWithOptions(baseURL, options...), nil
}

// Option customizes the client behavior (retries, TLS/mTLS, service identity).
type Option func(*Client)

// WithRetryConfig sets the retry configuration for the client.
func WithRetryConfig(cfg RetryConfig) Option {
	return func(c *Client) {
		c.retryCfg = cfg.normalize()
	}
}

// WithTLSConfig sets a TLS configuration for the underlying transport (supports mTLS).
func WithTLSConfig(cfg *tls.Config) Option {
	return func(c *Client) {
		c.tlsConfig = cfg
	}
}

// WithCircuitBreakerConfig sets circuit breaker parameters.
func WithCircuitBreakerConfig(cfg CircuitBreakerConfig) Option {
	return func(c *Client) {
		if cfg.FailureThreshold <= 0 {
			cfg.FailureThreshold = 5
		}
		if cfg.Cooldown <= 0 {
			cfg.Cooldown = 5 * time.Second
		}
		c.cbCfg = cfg
	}
}

// WithStaticBearerToken injects a static bearer token for internal service calls when no Authorization header is present.
func WithStaticBearerToken(token string) Option {
	return func(c *Client) {
		c.serviceBearer = token
	}
}

// WithStaticBearerTokenFromEnv reads a static bearer token from the provided environment variable name.
func WithStaticBearerTokenFromEnv(envVar string) Option {
	return func(c *Client) {
		if envVar == "" {
			return
		}
		if token := os.Getenv(envVar); token != "" {
			c.serviceBearer = token
		}
	}
}

// WithServiceIdentity sets service identity headers applied to each outbound request.
func WithServiceIdentity(name, instance string) Option {
	return func(c *Client) {
		headers := map[string]string{}
		if name != "" {
			headers[serviceNameHeader] = name
		}
		if instance != "" {
			headers[serviceInstanceHeader] = instance
		}
		c.serviceHeaders = headers
	}
}

// TLSConfigFromFiles builds a tls.Config using CA/cert/key files (mTLS when cert/key provided).
func TLSConfigFromFiles(caFile, certFile, keyFile string) (*tls.Config, error) {
	rootCAs, err := loadCertPool(caFile)
	if err != nil {
		return nil, err
	}

	var certificates []tls.Certificate
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, err
		}
		certificates = []tls.Certificate{cert}
	}

	return &tls.Config{
		RootCAs:      rootCAs,
		Certificates: certificates,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func loadCertPool(caFile string) (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if caFile == "" {
		return pool, nil
	}
	data, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	if ok := pool.AppendCertsFromPEM(data); !ok {
		return nil, fmt.Errorf("failed to append CA certs")
	}
	return pool, nil
}

// NewWithOptions creates a new auth-gateway client with optional configuration.
func NewWithOptions(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:        baseURL,
		retryCfg:       defaultRetryConfig(),
		cbCfg:          CircuitBreakerConfig{FailureThreshold: 5, Cooldown: 5 * time.Second},
		serviceHeaders: map[string]string{},
	}

	for _, opt := range opts {
		opt(c)
	}

	c.httpClient = c.buildHTTPClient()
	return c
}

func (c *Client) buildHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: wrapTransport(defaultTransport(c.tlsConfig), c.retryCfg, c.cbCfg, c.serviceHeaders, c.serviceBearer),
	}
}

// WithHTTPClient overrides the underlying http.Client (useful for tests).
func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	if httpClient != nil {
		httpClient.Transport = wrapTransport(httpClient.Transport, c.retryCfg, c.cbCfg, c.serviceHeaders, c.serviceBearer)
		c.httpClient = httpClient
		c.customHTTPClient = true
	}
	return c
}

// WithRetry updates the retry configuration on an existing client (rebuilds transport if not custom).
func (c *Client) WithRetry(cfg RetryConfig) *Client {
	c.retryCfg = cfg.normalize()
	if !c.customHTTPClient {
		c.httpClient = c.buildHTTPClient()
	}
	return c
}

// WithCircuitBreaker updates the circuit breaker configuration (rebuilds transport if not custom).
func (c *Client) WithCircuitBreaker(cfg CircuitBreakerConfig) *Client {
	c.cbCfg = cfg
	if c.cbCfg.FailureThreshold <= 0 {
		c.cbCfg.FailureThreshold = 5
	}
	if c.cbCfg.Cooldown <= 0 {
		c.cbCfg.Cooldown = 5 * time.Second
	}
	if !c.customHTTPClient {
		c.httpClient = c.buildHTTPClient()
	}
	return c
}

// WithTLS updates the TLS configuration (rebuilds transport if not custom).
func (c *Client) WithTLS(cfg *tls.Config) *Client {
	c.tlsConfig = cfg
	if !c.customHTTPClient {
		c.httpClient = c.buildHTTPClient()
	}
	return c
}

// WithTLSFiles loads CA/cert/key files into a tls.Config and applies it.
func (c *Client) WithTLSFiles(caFile, certFile, keyFile string) (*Client, error) {
	cfg, err := TLSConfigFromFiles(caFile, certFile, keyFile)
	if err != nil {
		return c, err
	}
	return c.WithTLS(cfg), nil
}

// WithServiceBearer sets a static bearer token used on outbound internal calls when the caller does not provide Authorization.
// The header is only applied if missing, preserving any end-user token a handler adds.
func (c *Client) WithServiceBearer(token string) *Client {
	c.serviceBearer = token
	if !c.customHTTPClient {
		c.httpClient = c.buildHTTPClient()
	}
	return c
}

// WithServiceBearerFromEnv loads a bearer token from the provided environment variable (no-op when empty).
func (c *Client) WithServiceBearerFromEnv(envVar string) *Client {
	if envVar == "" {
		return c
	}
	if token := os.Getenv(envVar); token != "" {
		return c.WithServiceBearer(token)
	}
	return c
}

// WithService sets outbound service identity headers (rebuilds transport if not custom).
func (c *Client) WithService(name, instance string) *Client {
	service := map[string]string{}
	if name != "" {
		service[serviceNameHeader] = name
	}
	if instance != "" {
		service[serviceInstanceHeader] = instance
	}
	c.serviceHeaders = service
	if !c.customHTTPClient {
		c.httpClient = c.buildHTTPClient()
	}
	return c
}

// HTTPError provides a structured error for non-2xx responses.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("unexpected status code: %d", e.StatusCode)
}

func readErrorBody(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return string(data)
}

// ValidateToken validates a JWT by calling auth-gateway.
func (c *Client) ValidateToken(token string) (*types.Claims, error) {
	return c.ValidateTokenWithContext(context.Background(), token)
}

// ValidateTokenWithContext validates a JWT by calling auth-gateway using the provided context.
func (c *Client) ValidateTokenWithContext(ctx context.Context, token string) (*types.Claims, error) {
	url := fmt.Sprintf("%s/api/v1/auth/validate", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, autherrors.ErrUnauthorized
	}

	if resp.StatusCode != http.StatusOK {
		body := readErrorBody(resp)
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: body}
	}

	var claims types.Claims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, err
	}

	return &claims, nil
}

// GetUserByID fetches a user by ID from auth-gateway.
func (c *Client) GetUserByID(userID, token string) (*types.User, error) {
	return c.GetUserByIDWithContext(context.Background(), userID, token)
}

// GetUserByIDWithContext fetches a user by ID using the provided context.
func (c *Client) GetUserByIDWithContext(ctx context.Context, userID, token string) (*types.User, error) {
	url := fmt.Sprintf("%s/api/v1/auth/users/%s", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, autherrors.ErrUserNotFound
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, autherrors.ErrUnauthorized
	}

	if resp.StatusCode != http.StatusOK {
		body := readErrorBody(resp)
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: body}
	}

	var user types.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetMe fetches the current user from auth-gateway using the provided token.
func (c *Client) GetMe(token string) (*types.User, error) {
	return c.GetMeWithContext(context.Background(), token)
}

// GetMeWithContext fetches the current user using the provided context.
func (c *Client) GetMeWithContext(ctx context.Context, token string) (*types.User, error) {
	url := fmt.Sprintf("%s/api/v1/auth/me", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, autherrors.ErrUnauthorized
	}

	if resp.StatusCode != http.StatusOK {
		body := readErrorBody(resp)
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: body}
	}

	var user types.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// RefreshToken renews the access token using the refresh token.
func (c *Client) RefreshToken(refreshToken string) (*types.TokenResponse, error) {
	return c.RefreshTokenWithContext(context.Background(), refreshToken)
}

// RefreshTokenWithContext renews the access token using the provided context.
func (c *Client) RefreshTokenWithContext(ctx context.Context, refreshToken string) (*types.TokenResponse, error) {
	url := fmt.Sprintf("%s/api/v1/auth/refresh", c.baseURL)

	payload := map[string]string{
		"refreshToken": refreshToken,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, autherrors.ErrUnauthorized
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tokens types.TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}

	return &tokens, nil
}

// Logout revokes the current session.
func (c *Client) Logout(token string) error {
	return c.LogoutWithContext(context.Background(), token)
}

// LogoutWithContext revokes the current session using the provided context.
func (c *Client) LogoutWithContext(ctx context.Context, token string) error {
	url := fmt.Sprintf("%s/api/v1/auth/logout", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return autherrors.ErrUnauthorized
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return &HTTPError{StatusCode: resp.StatusCode, Body: readErrorBody(resp)}
	}

	return nil
}
