package oauth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/usecase/auth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleProvider implements Google OAuth
type GoogleProvider struct {
	config   *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// NewGoogleProvider creates a new Google OAuth provider
func NewGoogleProvider(clientID, clientSecret, redirectURL string) (*GoogleProvider, error) {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, err
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     google.Endpoint,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	return &GoogleProvider{
		config:   config,
		verifier: verifier,
	}, nil
}

// GetAuthURL returns the Google OAuth authorization URL
func (p *GoogleProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges authorization code for user info
func (p *GoogleProvider) ExchangeCode(ctx context.Context, code string) (*auth.OAuthUserInfo, error) {
	// Exchange code for token
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Extract ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}

	// Verify ID token
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims
	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	return &auth.OAuthUserInfo{
		ProviderID: claims.Sub,
		Email:      claims.Email,
		Name:       claims.Name,
		Verified:   claims.EmailVerified,
	}, nil
}
