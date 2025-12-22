package client

import (
	"bytes"
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
)

// Client is an HTTP client used to communicate with auth-gateway.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new auth-gateway HTTP client.
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NewFromEnv creates a new client using EnvAuthGatewayURL and returns an error if missing.
func NewFromEnv() (*Client, error) {
	baseURL := os.Getenv(EnvAuthGatewayURL)
	if baseURL == "" {
		return nil, fmt.Errorf("environment variable %s not set", EnvAuthGatewayURL)
	}
	return New(baseURL), nil
}

// ValidateToken validates a JWT by calling auth-gateway.
func (c *Client) ValidateToken(token string) (*types.Claims, error) {
	url := fmt.Sprintf("%s/api/v1/auth/validate", c.baseURL)

	req, err := http.NewRequest(http.MethodGet, url, nil)
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
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var claims types.Claims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, err
	}

	return &claims, nil
}

// GetUserByID fetches a user by ID from auth-gateway.
func (c *Client) GetUserByID(userID, token string) (*types.User, error) {
	url := fmt.Sprintf("%s/api/v1/auth/users/%s", c.baseURL, userID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
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
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var user types.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetMe fetches the current user from auth-gateway using the provided token.
func (c *Client) GetMe(token string) (*types.User, error) {
	url := fmt.Sprintf("%s/api/v1/auth/me", c.baseURL)

	req, err := http.NewRequest(http.MethodGet, url, nil)
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
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var user types.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// RefreshToken renews the access token using the refresh token.
func (c *Client) RefreshToken(refreshToken string) (*types.TokenResponse, error) {
	url := fmt.Sprintf("%s/api/v1/auth/refresh", c.baseURL)

	payload := map[string]string{
		"refreshToken": refreshToken,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
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
	url := fmt.Sprintf("%s/api/v1/auth/logout", c.baseURL)

	req, err := http.NewRequest(http.MethodPost, url, nil)
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
