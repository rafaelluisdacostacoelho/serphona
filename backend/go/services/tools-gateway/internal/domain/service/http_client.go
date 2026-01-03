package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

// HTTPClient defines the interface for making HTTP requests
type HTTPClient interface {
	// Execute executes an HTTP request with the given configuration
	Execute(ctx context.Context, tool *entity.Tool, input map[string]interface{}, authConfig json.RawMessage) (*HTTPResponse, error)
}

// HTTPResponse represents the response from an HTTP request
type HTTPResponse struct {
	StatusCode int                    `json:"status_code"`
	Headers    map[string][]string    `json:"headers"`
	Body       map[string]interface{} `json:"body"`
	RawBody    []byte                 `json:"-"`
	LatencyMS  int64                  `json:"latency_ms"`
}

// httpClientImpl implements HTTPClient
type httpClientImpl struct {
	client          *http.Client
	audience        string
	serviceName     string
	serviceInstance string
}

// NewHTTPClient creates a new HTTPClient
func NewHTTPClient(timeout time.Duration, serviceName, serviceInstance, audience string) HTTPClient {
	return &httpClientImpl{
		client: &http.Client{
			Timeout: timeout,
		},
		audience:        audience,
		serviceName:     serviceName,
		serviceInstance: serviceInstance,
	}
}

// Execute executes an HTTP request with retry logic
func (c *httpClientImpl) Execute(ctx context.Context, tool *entity.Tool, input map[string]interface{}, authConfig json.RawMessage) (*HTTPResponse, error) {
	var response *HTTPResponse
	var lastErr error

	startTime := time.Now()

	// Retry configuration
	retryOpts := []retry.Option{
		retry.Attempts(uint(tool.MaxRetries)),
		retry.Delay(time.Duration(tool.RetryDelaySeconds) * time.Second),
		retry.Context(ctx),
		retry.LastErrorOnly(true),
		retry.RetryIf(func(err error) bool {
			// Retry on temporary errors
			return isRetryableError(err)
		}),
	}

	// Execute with retry
	err := retry.Do(
		func() error {
			resp, err := c.executeOnce(ctx, tool, input, authConfig)
			if err != nil {
				lastErr = err
				return err
			}
			response = resp
			return nil
		},
		retryOpts...,
	)

	if err != nil {
		return nil, fmt.Errorf("failed after %d retries: %w", tool.MaxRetries, lastErr)
	}

	// Calculate latency
	response.LatencyMS = time.Since(startTime).Milliseconds()

	return response, nil
}

// executeOnce executes a single HTTP request
func (c *httpClientImpl) executeOnce(ctx context.Context, tool *entity.Tool, input map[string]interface{}, authConfig json.RawMessage) (*HTTPResponse, error) {
	// Build URL
	url := tool.BaseURL + tool.EndpointPath

	// Replace path parameters
	url = c.replacePlaceholders(url, input)

	// Prepare request body
	var body io.Reader
	if tool.Method == entity.HTTPMethodPOST || tool.Method == entity.HTTPMethodPUT || tool.Method == entity.HTTPMethodPATCH {
		jsonBody, err := json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewReader(jsonBody)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, tool.Method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	if err := c.addHeaders(req, tool, authConfig); err != nil {
		return nil, err
	}

	// Add tenant header from context (platform-auth)
	if tenantID, err := middleware.TenantIDFromContext(ctx); err == nil && tenantID != "" {
		req.Header.Set(middleware.TenantIDHeader, tenantID)
	}

	// Add service identity headers for internal tracing/auth
	c.addServiceIdentity(req)

	// Add query parameters for GET requests
	if tool.Method == entity.HTTPMethodGET {
		c.addQueryParams(req, input)
	}

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	response := &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		RawBody:    rawBody,
	}

	// Try to parse JSON body
	if len(rawBody) > 0 {
		var jsonBody map[string]interface{}
		if err := json.Unmarshal(rawBody, &jsonBody); err == nil {
			response.Body = jsonBody
		}
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return response, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(rawBody))
	}

	return response, nil
}

// addHeaders adds headers to the request including authentication
func (c *httpClientImpl) addHeaders(req *http.Request, tool *entity.Tool, authConfig json.RawMessage) error {
	// Add tool headers
	var headers map[string]string
	if len(tool.Headers) > 0 {
		if err := json.Unmarshal(tool.Headers, &headers); err != nil {
			return fmt.Errorf("failed to unmarshal headers: %w", err)
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	// Add authentication
	if err := c.addAuthentication(req, tool.AuthType, tool.AuthConfig, authConfig); err != nil {
		return err
	}

	// Set default content type if not set
	if req.Header.Get("Content-Type") == "" && req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return nil
}

// addServiceIdentity injects service identity headers when configured.
func (c *httpClientImpl) addServiceIdentity(req *http.Request) {
	if c.serviceName != "" {
		req.Header.Set("X-Service-Name", c.serviceName)
	}
	if c.serviceInstance != "" {
		req.Header.Set("X-Service-Instance", c.serviceInstance)
	}
}

// addAuthentication adds authentication headers/params based on auth type
func (c *httpClientImpl) addAuthentication(req *http.Request, authType string, defaultConfig, customConfig json.RawMessage) error {
	// Use custom config if provided, otherwise use default
	config := defaultConfig
	if len(customConfig) > 0 {
		config = customConfig
	}

	switch authType {
	case entity.AuthTypeAPIKey.String():
		return c.addAPIKeyAuth(req, config)
	case entity.AuthTypeBearer.String():
		return c.addBearerAuth(req, config)
	case entity.AuthTypeBasic.String():
		return c.addBasicAuth(req, config)
	case entity.AuthTypeNone.String():
		return nil
	default:
		return fmt.Errorf("unsupported auth type: %s", authType)
	}
}

// addAPIKeyAuth adds API key authentication
func (c *httpClientImpl) addAPIKeyAuth(req *http.Request, config json.RawMessage) error {
	var authConfig struct {
		APIKey        string `json:"api_key"`
		ParamName     string `json:"param_name"`
		ParamLocation string `json:"param_location"` // query, header
		HeaderName    string `json:"header_name"`
	}

	if err := json.Unmarshal(config, &authConfig); err != nil {
		return fmt.Errorf("failed to parse API key config: %w", err)
	}

	if authConfig.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	switch authConfig.ParamLocation {
	case "header":
		headerName := authConfig.HeaderName
		if headerName == "" {
			headerName = "X-API-Key"
		}
		req.Header.Set(headerName, authConfig.APIKey)
	case "query":
		paramName := authConfig.ParamName
		if paramName == "" {
			paramName = "api_key"
		}
		q := req.URL.Query()
		q.Set(paramName, authConfig.APIKey)
		req.URL.RawQuery = q.Encode()
	default:
		return fmt.Errorf("invalid param_location: %s", authConfig.ParamLocation)
	}

	return nil
}

// addBearerAuth adds Bearer token authentication
func (c *httpClientImpl) addBearerAuth(req *http.Request, config json.RawMessage) error {
	var authConfig struct {
		Token      string `json:"token"`
		HeaderName string `json:"header_name"`
		Prefix     string `json:"prefix"`
	}

	if err := json.Unmarshal(config, &authConfig); err != nil {
		return fmt.Errorf("failed to parse bearer config: %w", err)
	}

	if authConfig.Token == "" {
		return fmt.Errorf("bearer token is required")
	}

	headerName := authConfig.HeaderName
	if headerName == "" {
		headerName = "Authorization"
	}

	prefix := authConfig.Prefix
	if prefix == "" {
		prefix = "Bearer"
	}

	if strings.EqualFold(prefix, "bearer") && c.audience != "" {
		if err := validateAudience(authConfig.Token, c.audience); err != nil {
			return err
		}
	}

	req.Header.Set(headerName, fmt.Sprintf("%s %s", prefix, authConfig.Token))
	return nil
}

// addBasicAuth adds Basic authentication
func (c *httpClientImpl) addBasicAuth(req *http.Request, config json.RawMessage) error {
	var authConfig struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.Unmarshal(config, &authConfig); err != nil {
		return fmt.Errorf("failed to parse basic auth config: %w", err)
	}

	if authConfig.Username == "" || authConfig.Password == "" {
		return fmt.Errorf("username and password are required")
	}

	req.SetBasicAuth(authConfig.Username, authConfig.Password)
	return nil
}

// addQueryParams adds query parameters to the request
func (c *httpClientImpl) addQueryParams(req *http.Request, params map[string]interface{}) {
	q := req.URL.Query()
	for key, value := range params {
		q.Set(key, fmt.Sprintf("%v", value))
	}
	req.URL.RawQuery = q.Encode()
}

// replacePlaceholders replaces placeholders in URL with values from input
func (c *httpClientImpl) replacePlaceholders(url string, input map[string]interface{}) string {
	for key, value := range input {
		placeholder := fmt.Sprintf("{%s}", key)
		if strings.Contains(url, placeholder) {
			url = strings.ReplaceAll(url, placeholder, fmt.Sprintf("%v", value))
		}
	}
	return url
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retry on temporary network errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "temporary failure") {
		return true
	}

	// Retry on 5xx errors (except 501)
	if strings.Contains(errStr, "HTTP 500") ||
		strings.Contains(errStr, "HTTP 502") ||
		strings.Contains(errStr, "HTTP 503") ||
		strings.Contains(errStr, "HTTP 504") {
		return true
	}

	// Retry on 429 (rate limit)
	if strings.Contains(errStr, "HTTP 429") {
		return true
	}

	return false
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

	return fmt.Errorf("service token audience mismatch")
}
