package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/usecase/auth"
	"golang.org/x/oauth2"
)

// AppleProvider implements Apple OAuth (Sign in with Apple)
type AppleProvider struct {
	config *oauth2.Config
}

// NewAppleProvider creates a new Apple OAuth provider
func NewAppleProvider(clientID, teamID, keyID, privateKey, redirectURL string) (*AppleProvider, error) {
	// Apple uses a custom endpoint
	config := &oauth2.Config{
		ClientID:    clientID,
		RedirectURL: redirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://appleid.apple.com/auth/authorize",
			TokenURL: "https://appleid.apple.com/auth/token",
		},
		Scopes: []string{"name", "email"},
	}

	// Note: Apple requires a JWT as client secret, which would be generated using the teamID, keyID, and privateKey
	// For simplicity, we're showing the structure. In production, you'd generate the JWT client secret.

	return &AppleProvider{
		config: config,
	}, nil
}

// GetAuthURL returns the Apple OAuth authorization URL
func (p *AppleProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.SetAuthURLParam("response_mode", "form_post"))
}

// ExchangeCode exchanges authorization code for user info
func (p *AppleProvider) ExchangeCode(ctx context.Context, code string) (*auth.OAuthUserInfo, error) {
	// Exchange code for token
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Apple returns user info in the id_token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}

	// Parse ID token (simplified - in production use a proper JWT library)
	userInfo, err := parseAppleIDToken(rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID token: %w", err)
	}

	return userInfo, nil
}

// parseAppleIDToken parses Apple ID token
// Note: This is a simplified version. In production, properly verify the JWT signature
func parseAppleIDToken(idToken string) (*auth.OAuthUserInfo, error) {
	// In production, use a proper JWT library to decode and verify
	// For now, we'll make a request to Apple's userinfo endpoint
	req, err := http.NewRequest("GET", "https://appleid.apple.com/auth/userinfo", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+idToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var claims struct {
		Sub            string `json:"sub"`
		Email          string `json:"email"`
		EmailVerified  string `json:"email_verified"`
		IsPrivateEmail string `json:"is_private_email"`
	}

	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, err
	}

	return &auth.OAuthUserInfo{
		ProviderID: claims.Sub,
		Email:      claims.Email,
		Name:       claims.Email, // Apple doesn't always provide name
		Verified:   claims.EmailVerified == "true",
	}, nil
}
