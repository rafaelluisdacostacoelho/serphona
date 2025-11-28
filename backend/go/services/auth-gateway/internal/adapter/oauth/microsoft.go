package oauth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/usecase/auth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

// MicrosoftProvider implements Microsoft OAuth
type MicrosoftProvider struct {
	config   *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// NewMicrosoftProvider creates a new Microsoft OAuth provider
func NewMicrosoftProvider(clientID, clientSecret, redirectURL string) (*MicrosoftProvider, error) {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, "https://login.microsoftonline.com/common/v2.0")
	if err != nil {
		return nil, err
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     microsoft.AzureADEndpoint("common"),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	return &MicrosoftProvider{
		config:   config,
		verifier: verifier,
	}, nil
}

// GetAuthURL returns the Microsoft OAuth authorization URL
func (p *MicrosoftProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

// ExchangeCode exchanges authorization code for user info
func (p *MicrosoftProvider) ExchangeCode(ctx context.Context, code string) (*auth.OAuthUserInfo, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	return &auth.OAuthUserInfo{
		ProviderID: claims.Sub,
		Email:      claims.Email,
		Name:       claims.Name,
		Verified:   true, // Microsoft emails are verified
	}, nil
}
