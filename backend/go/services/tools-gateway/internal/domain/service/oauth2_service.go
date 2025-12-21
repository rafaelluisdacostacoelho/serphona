// Package service contains domain services.
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

// OAuth2Service handles OAuth 2.0 flows.
type OAuth2Service interface {
	// GenerateAuthorizationURL generates the OAuth authorization URL for user redirect.
	GenerateAuthorizationURL(
		ctx context.Context,
		integration *entity.Integration,
		tenantID, userID uuid.UUID,
		scopes []string,
		usePKCE bool,
	) (authURL string, state *entity.OAuthState, err error)

	// ExchangeCodeForToken exchanges authorization code for access token.
	ExchangeCodeForToken(
		ctx context.Context,
		integration *entity.Integration,
		code string,
		state *entity.OAuthState,
	) (*entity.OAuthToken, error)

	// RefreshAccessToken refreshes an expired access token.
	RefreshAccessToken(
		ctx context.Context,
		integration *entity.Integration,
		token *entity.OAuthToken,
	) (*entity.OAuthToken, error)

	// GetClientCredentialsToken gets a token using client credentials grant.
	GetClientCredentialsToken(
		ctx context.Context,
		integration *entity.Integration,
		tenantID uuid.UUID,
		scopes []string,
	) (*entity.OAuthToken, error)

	// RevokeToken revokes an access or refresh token.
	RevokeToken(
		ctx context.Context,
		integration *entity.Integration,
		token *entity.OAuthToken,
	) error
}

// oauth2ServiceImpl implements OAuth2Service.
type oauth2ServiceImpl struct {
	httpClient *http.Client
}

// NewOAuth2Service creates a new OAuth2Service.
func NewOAuth2Service(timeout time.Duration) OAuth2Service {
	return &oauth2ServiceImpl{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GenerateAuthorizationURL generates the OAuth authorization URL.
func (s *oauth2ServiceImpl) GenerateAuthorizationURL(
	ctx context.Context,
	integration *entity.Integration,
	tenantID, userID uuid.UUID,
	scopes []string,
	usePKCE bool,
) (string, *entity.OAuthState, error) {
	if integration.OAuth2Config == nil {
		return "", nil, fmt.Errorf("OAuth2 config is required")
	}

	config := integration.OAuth2Config

	// Generate random state for CSRF protection
	stateValue, err := generateRandomString(32)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Create OAuth state
	oauthState := &entity.OAuthState{
		ID:            uuid.New(),
		IntegrationID: integration.ID,
		TenantID:      tenantID,
		UserID:        userID,
		State:         stateValue,
		RedirectURI:   config.RedirectURL,
		Scopes:        scopes,
		ExpiresAt:     time.Now().UTC().Add(10 * time.Minute), // State expires in 10 minutes
	}

	// Build authorization URL
	params := url.Values{}
	params.Set("client_id", config.ClientID)
	params.Set("redirect_uri", config.RedirectURL)
	params.Set("response_type", "code")
	params.Set("state", stateValue)

	// Add scopes
	if len(scopes) > 0 {
		params.Set("scope", strings.Join(scopes, " "))
	} else if len(config.Scopes) > 0 {
		params.Set("scope", strings.Join(config.Scopes, " "))
	}

	// PKCE support
	if usePKCE {
		codeVerifier, err := generateRandomString(64)
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate code verifier: %w", err)
		}
		oauthState.CodeVerifier = codeVerifier

		// Generate code challenge
		codeChallenge := generateCodeChallenge(codeVerifier)
		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", "S256")
	}

	// Add additional params
	if config.AdditionalParams != nil {
		for key, value := range config.AdditionalParams {
			params.Set(key, value)
		}
	}

	authURL := config.AuthURL + "?" + params.Encode()

	return authURL, oauthState, nil
}

// ExchangeCodeForToken exchanges authorization code for access token.
func (s *oauth2ServiceImpl) ExchangeCodeForToken(
	ctx context.Context,
	integration *entity.Integration,
	code string,
	state *entity.OAuthState,
) (*entity.OAuthToken, error) {
	if integration.OAuth2Config == nil {
		return nil, fmt.Errorf("OAuth2 config is required")
	}

	config := integration.OAuth2Config

	// Prepare token request
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", state.RedirectURI)
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)

	// Add code verifier for PKCE
	if state.CodeVerifier != "" {
		data.Set("code_verifier", state.CodeVerifier)
	}

	// Execute token request
	tokenResp, err := s.executeTokenRequest(ctx, config.TokenURL, data, config)
	if err != nil {
		return nil, err
	}

	// Create OAuth token
	token := s.createTokenFromResponse(tokenResp, integration.ID, state.TenantID, &state.UserID)

	return token, nil
}

// RefreshAccessToken refreshes an expired access token.
func (s *oauth2ServiceImpl) RefreshAccessToken(
	ctx context.Context,
	integration *entity.Integration,
	token *entity.OAuthToken,
) (*entity.OAuthToken, error) {
	if integration.OAuth2Config == nil {
		return nil, fmt.Errorf("OAuth2 config is required")
	}

	if token.RefreshToken == "" {
		return nil, fmt.Errorf("refresh token is not available")
	}

	config := integration.OAuth2Config

	// Prepare refresh request
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", token.RefreshToken)
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)

	// Add scopes if available
	if len(token.Scopes) > 0 {
		data.Set("scope", strings.Join(token.Scopes, " "))
	}

	// Execute token request
	tokenResp, err := s.executeTokenRequest(ctx, config.TokenURL, data, config)
	if err != nil {
		return nil, err
	}

	// Create new token (keep refresh token if not returned)
	newToken := s.createTokenFromResponse(tokenResp, integration.ID, token.TenantID, token.UserID)
	if newToken.RefreshToken == "" {
		newToken.RefreshToken = token.RefreshToken
	}

	return newToken, nil
}

// GetClientCredentialsToken gets a token using client credentials grant.
func (s *oauth2ServiceImpl) GetClientCredentialsToken(
	ctx context.Context,
	integration *entity.Integration,
	tenantID uuid.UUID,
	scopes []string,
) (*entity.OAuthToken, error) {
	if integration.OAuth2Config == nil {
		return nil, fmt.Errorf("OAuth2 config is required")
	}

	config := integration.OAuth2Config

	// Prepare token request
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)

	// Add scopes
	if len(scopes) > 0 {
		data.Set("scope", strings.Join(scopes, " "))
	} else if len(config.Scopes) > 0 {
		data.Set("scope", strings.Join(config.Scopes, " "))
	}

	// Execute token request
	tokenResp, err := s.executeTokenRequest(ctx, config.TokenURL, data, config)
	if err != nil {
		return nil, err
	}

	// Create OAuth token (no user for client credentials)
	token := s.createTokenFromResponse(tokenResp, integration.ID, tenantID, nil)

	return token, nil
}

// RevokeToken revokes an access or refresh token.
func (s *oauth2ServiceImpl) RevokeToken(
	ctx context.Context,
	integration *entity.Integration,
	token *entity.OAuthToken,
) error {
	if integration.OAuth2Config == nil {
		return fmt.Errorf("OAuth2 config is required")
	}

	// Note: Revocation endpoint is optional and provider-specific
	// This is a placeholder - implement based on provider requirements

	// Mark token as revoked locally
	token.Revoke()

	return nil
}

// executeTokenRequest executes a token exchange request.
func (s *oauth2ServiceImpl) executeTokenRequest(
	ctx context.Context,
	tokenURL string,
	data url.Values,
	config *entity.OAuth2Config,
) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Handle authentication method
	switch config.TokenEndpointAuthMethod {
	case "client_secret_basic", "":
		// Default: Basic auth
		req.SetBasicAuth(config.ClientID, config.ClientSecret)
	case "client_secret_post":
		// Credentials already in POST body
	default:
		return nil, fmt.Errorf("unsupported token endpoint auth method: %s", config.TokenEndpointAuthMethod)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp map[string]interface{}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return tokenResp, nil
}

// createTokenFromResponse creates an OAuthToken from the provider response.
func (s *oauth2ServiceImpl) createTokenFromResponse(
	resp map[string]interface{},
	integrationID, tenantID uuid.UUID,
	userID *uuid.UUID,
) *entity.OAuthToken {
	token := &entity.OAuthToken{
		ID:            uuid.New(),
		IntegrationID: integrationID,
		TenantID:      tenantID,
		UserID:        userID,
		AccessToken:   getString(resp, "access_token"),
		RefreshToken:  getString(resp, "refresh_token"),
		TokenType:     getString(resp, "token_type"),
		IsValid:       true,
		IsRevoked:     false,
		Extra:         make(map[string]interface{}),
	}

	// Set token type default
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}

	// Parse expiration
	if expiresIn, ok := resp["expires_in"].(float64); ok {
		expiresAt := time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
		token.ExpiresAt = &expiresAt

		// Set refresh time to 5 minutes before expiration
		refreshAt := expiresAt.Add(-5 * time.Minute)
		token.RefreshAt = &refreshAt
	}

	// Parse scopes
	if scopeStr, ok := resp["scope"].(string); ok {
		token.Scopes = strings.Split(scopeStr, " ")
	}

	// Store extra fields
	for key, value := range resp {
		if key != "access_token" && key != "refresh_token" && key != "token_type" &&
			key != "expires_in" && key != "scope" {
			token.Extra[key] = value
		}
	}

	return token
}

// Helper functions

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
