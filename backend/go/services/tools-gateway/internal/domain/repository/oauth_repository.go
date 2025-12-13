// Package repository contains repository interfaces.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

// OAuthTokenRepository defines the interface for OAuth token data access.
type OAuthTokenRepository interface {
	// Create creates a new OAuth token.
	Create(ctx context.Context, token *entity.OAuthToken) error

	// FindByID finds a token by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.OAuthToken, error)

	// FindByIntegrationAndTenant finds a token by integration and tenant.
	FindByIntegrationAndTenant(ctx context.Context, integrationID, tenantID uuid.UUID) (*entity.OAuthToken, error)

	// FindByIntegrationAndUser finds a token by integration and user.
	FindByIntegrationAndUser(ctx context.Context, integrationID, tenantID, userID uuid.UUID) (*entity.OAuthToken, error)

	// FindValidToken finds a valid (not expired, not revoked) token.
	FindValidToken(ctx context.Context, integrationID, tenantID uuid.UUID, userID *uuid.UUID) (*entity.OAuthToken, error)

	// FindTokensNeedingRefresh finds tokens that need to be refreshed.
	FindTokensNeedingRefresh(ctx context.Context) ([]*entity.OAuthToken, error)

	// Update updates a token.
	Update(ctx context.Context, token *entity.OAuthToken) error

	// Delete deletes a token.
	Delete(ctx context.Context, id uuid.UUID) error

	// RevokeByIntegration revokes all tokens for an integration.
	RevokeByIntegration(ctx context.Context, integrationID uuid.UUID) error

	// DeleteExpired deletes expired tokens (cleanup).
	DeleteExpired(ctx context.Context) error
}

// OAuthStateRepository defines the interface for OAuth state data access.
type OAuthStateRepository interface {
	// Create creates a new OAuth state.
	Create(ctx context.Context, state *entity.OAuthState) error

	// FindByState finds a state by state value.
	FindByState(ctx context.Context, state string) (*entity.OAuthState, error)

	// MarkAsUsed marks a state as used.
	MarkAsUsed(ctx context.Context, id uuid.UUID) error

	// DeleteExpired deletes expired states (cleanup).
	DeleteExpired(ctx context.Context) error
}
