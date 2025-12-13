// Package postgres contains PostgreSQL repository implementations.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/domain/repository"
	"gorm.io/gorm"
)

// oauthTokenRepositoryImpl implements repository.OAuthTokenRepository.
type oauthTokenRepositoryImpl struct {
	db *gorm.DB
}

// NewOAuthTokenRepository creates a new OAuthTokenRepository.
func NewOAuthTokenRepository(db *gorm.DB) repository.OAuthTokenRepository {
	return &oauthTokenRepositoryImpl{db: db}
}

// Create creates a new OAuth token.
func (r *oauthTokenRepositoryImpl) Create(ctx context.Context, token *entity.OAuthToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("failed to create token: %w", err)
	}
	return nil
}

// FindByID finds a token by ID.
func (r *oauthTokenRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.OAuthToken, error) {
	var token entity.OAuthToken
	if err := r.db.WithContext(ctx).First(&token, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("failed to find token: %w", err)
	}
	return &token, nil
}

// FindByIntegrationAndTenant finds a token by integration and tenant.
func (r *oauthTokenRepositoryImpl) FindByIntegrationAndTenant(ctx context.Context, integrationID, tenantID uuid.UUID) (*entity.OAuthToken, error) {
	var token entity.OAuthToken
	if err := r.db.WithContext(ctx).
		Where("integration_id = ? AND tenant_id = ? AND user_id IS NULL", integrationID, tenantID).
		Order("created_at DESC").
		First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("failed to find token: %w", err)
	}
	return &token, nil
}

// FindByIntegrationAndUser finds a token by integration and user.
func (r *oauthTokenRepositoryImpl) FindByIntegrationAndUser(ctx context.Context, integrationID, tenantID, userID uuid.UUID) (*entity.OAuthToken, error) {
	var token entity.OAuthToken
	if err := r.db.WithContext(ctx).
		Where("integration_id = ? AND tenant_id = ? AND user_id = ?", integrationID, tenantID, userID).
		Order("created_at DESC").
		First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("failed to find token: %w", err)
	}
	return &token, nil
}

// FindValidToken finds a valid (not expired, not revoked) token.
func (r *oauthTokenRepositoryImpl) FindValidToken(ctx context.Context, integrationID, tenantID uuid.UUID, userID *uuid.UUID) (*entity.OAuthToken, error) {
	query := r.db.WithContext(ctx).
		Where("integration_id = ? AND tenant_id = ? AND is_valid = ? AND is_revoked = ?",
			integrationID, tenantID, true, false)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	// Also check expiration
	query = query.Where("(expires_at IS NULL OR expires_at > ?)", time.Now().UTC())

	var token entity.OAuthToken
	if err := query.Order("created_at DESC").First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no valid token found")
		}
		return nil, fmt.Errorf("failed to find valid token: %w", err)
	}

	return &token, nil
}

// FindTokensNeedingRefresh finds tokens that need to be refreshed.
func (r *oauthTokenRepositoryImpl) FindTokensNeedingRefresh(ctx context.Context) ([]*entity.OAuthToken, error) {
	var tokens []*entity.OAuthToken
	now := time.Now().UTC()

	if err := r.db.WithContext(ctx).
		Where("is_valid = ? AND is_revoked = ? AND refresh_token IS NOT NULL AND refresh_token != ''",
			true, false).
		Where("refresh_at IS NOT NULL AND refresh_at <= ?", now).
		Find(&tokens).Error; err != nil {
		return nil, fmt.Errorf("failed to find tokens needing refresh: %w", err)
	}

	return tokens, nil
}

// Update updates a token.
func (r *oauthTokenRepositoryImpl) Update(ctx context.Context, token *entity.OAuthToken) error {
	if err := r.db.WithContext(ctx).Save(token).Error; err != nil {
		return fmt.Errorf("failed to update token: %w", err)
	}
	return nil
}

// Delete deletes a token.
func (r *oauthTokenRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&entity.OAuthToken{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}
	return nil
}

// RevokeByIntegration revokes all tokens for an integration.
func (r *oauthTokenRepositoryImpl) RevokeByIntegration(ctx context.Context, integrationID uuid.UUID) error {
	now := time.Now().UTC()
	if err := r.db.WithContext(ctx).
		Model(&entity.OAuthToken{}).
		Where("integration_id = ?", integrationID).
		Updates(map[string]interface{}{
			"is_valid":   false,
			"is_revoked": true,
			"revoked_at": now,
		}).Error; err != nil {
		return fmt.Errorf("failed to revoke tokens: %w", err)
	}
	return nil
}

// DeleteExpired deletes expired tokens.
func (r *oauthTokenRepositoryImpl) DeleteExpired(ctx context.Context) error {
	// Delete tokens expired more than 30 days ago
	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)

	if err := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", cutoff).
		Delete(&entity.OAuthToken{}).Error; err != nil {
		return fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	return nil
}

// oauthStateRepositoryImpl implements repository.OAuthStateRepository.
type oauthStateRepositoryImpl struct {
	db *gorm.DB
}

// NewOAuthStateRepository creates a new OAuthStateRepository.
func NewOAuthStateRepository(db *gorm.DB) repository.OAuthStateRepository {
	return &oauthStateRepositoryImpl{db: db}
}

// Create creates a new OAuth state.
func (r *oauthStateRepositoryImpl) Create(ctx context.Context, state *entity.OAuthState) error {
	if err := r.db.WithContext(ctx).Create(state).Error; err != nil {
		return fmt.Errorf("failed to create state: %w", err)
	}
	return nil
}

// FindByState finds a state by state value.
func (r *oauthStateRepositoryImpl) FindByState(ctx context.Context, state string) (*entity.OAuthState, error) {
	var oauthState entity.OAuthState
	if err := r.db.WithContext(ctx).First(&oauthState, "state = ?", state).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("state not found")
		}
		return nil, fmt.Errorf("failed to find state: %w", err)
	}
	return &oauthState, nil
}

// MarkAsUsed marks a state as used.
func (r *oauthStateRepositoryImpl) MarkAsUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	if err := r.db.WithContext(ctx).
		Model(&entity.OAuthState{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_used": true,
			"used_at": now,
		}).Error; err != nil {
		return fmt.Errorf("failed to mark state as used: %w", err)
	}
	return nil
}

// DeleteExpired deletes expired states.
func (r *oauthStateRepositoryImpl) DeleteExpired(ctx context.Context) error {
	now := time.Now().UTC()

	if err := r.db.WithContext(ctx).
		Where("expires_at < ?", now).
		Delete(&entity.OAuthState{}).Error; err != nil {
		return fmt.Errorf("failed to delete expired states: %w", err)
	}

	return nil
}
