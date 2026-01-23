// Package usecase contains application use cases.
package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/repository"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/service"
)

// OAuthService orchestrates OAuth 2.0 flows.
type OAuthService interface {
	// InitiateAuthFlow starts the OAuth authorization flow.
	InitiateAuthFlow(
		ctx context.Context,
		integrationID, tenantID, userID uuid.UUID,
		scopes []string,
		usePKCE bool,
	) (authURL string, err error)

	// HandleCallback processes the OAuth callback.
	HandleCallback(ctx context.Context, state, code string) (*entity.OAuthToken, error)

	// GetValidToken gets a valid token for an integration and user.
	GetValidToken(
		ctx context.Context,
		integrationID, tenantID uuid.UUID,
		userID *uuid.UUID,
	) (*entity.OAuthToken, error)

	// RefreshTokenIfNeeded refreshes a token if it needs refresh.
	RefreshTokenIfNeeded(ctx context.Context, token *entity.OAuthToken) (*entity.OAuthToken, error)

	// RevokeToken revokes a token.
	RevokeToken(ctx context.Context, tenantID uuid.UUID, tokenID uuid.UUID) error
}

// oauthServiceImpl implements OAuthService.
type oauthServiceImpl struct {
	integrationRepo repository.IntegrationRepository
	tokenRepo       repository.OAuthTokenRepository
	stateRepo       repository.OAuthStateRepository
	oauth2Service   service.OAuth2Service
}

// NewOAuthService creates a new OAuthService.
func NewOAuthService(
	integrationRepo repository.IntegrationRepository,
	tokenRepo repository.OAuthTokenRepository,
	stateRepo repository.OAuthStateRepository,
	oauth2Service service.OAuth2Service,
) OAuthService {
	return &oauthServiceImpl{
		integrationRepo: integrationRepo,
		tokenRepo:       tokenRepo,
		stateRepo:       stateRepo,
		oauth2Service:   oauth2Service,
	}
}

// InitiateAuthFlow starts the OAuth authorization flow.
func (s *oauthServiceImpl) InitiateAuthFlow(
	ctx context.Context,
	integrationID, tenantID, userID uuid.UUID,
	scopes []string,
	usePKCE bool,
) (string, error) {
	// Get integration
	integration, err := s.integrationRepo.FindByID(ctx, integrationID)
	if err != nil {
		return "", fmt.Errorf("integration not found: %w", err)
	}

	if integration.TenantID != tenantID {
		return "", fmt.Errorf("integration not found")
	}

	// Validate integration
	if !integration.IsActive {
		return "", fmt.Errorf("integration is not active")
	}

	if integration.AuthType != entity.AuthTypeOAuth2 {
		return "", fmt.Errorf("integration does not use OAuth2")
	}

	// Generate authorization URL
	authURL, state, err := s.oauth2Service.GenerateAuthorizationURL(
		ctx,
		integration,
		tenantID,
		userID,
		scopes,
		usePKCE,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate auth URL: %w", err)
	}

	// Save state
	if err := s.stateRepo.Create(ctx, state); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	return authURL, nil
}

// HandleCallback processes the OAuth callback.
func (s *oauthServiceImpl) HandleCallback(ctx context.Context, stateValue, code string) (*entity.OAuthToken, error) {
	// Find state
	state, err := s.stateRepo.FindByState(ctx, stateValue)
	if err != nil {
		return nil, fmt.Errorf("invalid state: %w", err)
	}

	// Validate state
	if !state.IsValid() {
		return nil, fmt.Errorf("state is expired or already used")
	}

	// Get integration
	integration, err := s.integrationRepo.FindByID(ctx, state.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	// Exchange code for token
	token, err := s.oauth2Service.ExchangeCodeForToken(ctx, integration, code, state)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Save token
	if err := s.tokenRepo.Create(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	// Mark state as used
	if err := s.stateRepo.MarkAsUsed(ctx, state.ID); err != nil {
		// Log error but don't fail the flow
		fmt.Printf("warning: failed to mark state as used: %v\n", err)
	}

	return token, nil
}

// GetValidToken gets a valid token for an integration and user.
func (s *oauthServiceImpl) GetValidToken(
	ctx context.Context,
	integrationID, tenantID uuid.UUID,
	userID *uuid.UUID,
) (*entity.OAuthToken, error) {
	// Try to find valid token
	token, err := s.tokenRepo.FindValidToken(ctx, integrationID, tenantID, userID)
	if err != nil {
		// No valid token found
		return nil, fmt.Errorf("no valid token found: user needs to authenticate")
	}

	// Check if token needs refresh
	if token.NeedsRefresh() {
		refreshedToken, err := s.RefreshTokenIfNeeded(ctx, token)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
		return refreshedToken, nil
	}

	return token, nil
}

// RefreshTokenIfNeeded refreshes a token if it needs refresh.
func (s *oauthServiceImpl) RefreshTokenIfNeeded(ctx context.Context, token *entity.OAuthToken) (*entity.OAuthToken, error) {
	// Check if refresh is needed
	if !token.NeedsRefresh() {
		return token, nil
	}

	// Get integration
	integration, err := s.integrationRepo.FindByID(ctx, token.IntegrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if integration.TenantID != token.TenantID {
		return nil, fmt.Errorf("integration not found")
	}

	// Refresh token
	newToken, err := s.oauth2Service.RefreshAccessToken(ctx, integration, token)
	if err != nil {
		// Mark old token as invalid
		token.IsValid = false
		_ = s.tokenRepo.Update(ctx, token)
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Save new token
	if err := s.tokenRepo.Create(ctx, newToken); err != nil {
		return nil, fmt.Errorf("failed to save refreshed token: %w", err)
	}

	// Mark old token as invalid
	token.IsValid = false
	if err := s.tokenRepo.Update(ctx, token); err != nil {
		// Log error but don't fail
		fmt.Printf("warning: failed to invalidate old token: %v\n", err)
	}

	return newToken, nil
}

// RevokeToken revokes a token.
func (s *oauthServiceImpl) RevokeToken(ctx context.Context, tenantID uuid.UUID, tokenID uuid.UUID) error {
	// Get token
	token, err := s.tokenRepo.FindByID(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("token not found: %w", err)
	}

	if token.TenantID != tenantID {
		return fmt.Errorf("token not found")
	}

	// Get integration
	integration, err := s.integrationRepo.FindByID(ctx, token.IntegrationID)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	if integration.TenantID != tenantID {
		return fmt.Errorf("integration not found")
	}

	// Revoke token at provider (if supported)
	if err := s.oauth2Service.RevokeToken(ctx, integration, token); err != nil {
		// Log error but continue with local revocation
		fmt.Printf("warning: failed to revoke token at provider: %v\n", err)
	}

	// Update token in database
	if err := s.tokenRepo.Update(ctx, token); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}
