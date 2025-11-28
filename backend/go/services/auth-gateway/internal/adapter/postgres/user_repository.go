package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/domain/user"
	"gorm.io/gorm"
)

// UserRepository implements user.Repository using PostgreSQL
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new PostgreSQL user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL", email).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByProvider retrieves a user by OAuth provider
func (r *UserRepository) GetByProvider(ctx context.Context, provider, providerID string) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_id = ? AND deleted_at IS NULL", provider, providerID).
		First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

// Delete soft deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&user.User{}).
		Where("id = ?", id).
		Update("deleted_at", time.Now()).Error
}

// CreateSession creates a new session
func (r *UserRepository) CreateSession(ctx context.Context, session *user.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

// GetSession retrieves a session by refresh token
func (r *UserRepository) GetSession(ctx context.Context, refreshToken string) (*user.Session, error) {
	var session user.Session
	err := r.db.WithContext(ctx).
		Where("refresh_token = ? AND revoked_at IS NULL", refreshToken).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// RevokeSession revokes a session
func (r *UserRepository) RevokeSession(ctx context.Context, refreshToken string) error {
	return r.db.WithContext(ctx).
		Model(&user.Session{}).
		Where("refresh_token = ?", refreshToken).
		Update("revoked_at", time.Now()).Error
}

// RevokeAllUserSessions revokes all sessions for a user
func (r *UserRepository) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&user.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

// CleanupExpiredSessions removes expired sessions
func (r *UserRepository) CleanupExpiredSessions(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ? OR revoked_at IS NOT NULL", time.Now().Add(-30*24*time.Hour)).
		Delete(&user.Session{}).Error
}

// CreateOAuthState creates a new OAuth state
func (r *UserRepository) CreateOAuthState(ctx context.Context, state *user.OAuthState) error {
	return r.db.WithContext(ctx).Create(state).Error
}

// GetOAuthState retrieves an OAuth state
func (r *UserRepository) GetOAuthState(ctx context.Context, stateStr string) (*user.OAuthState, error) {
	var state user.OAuthState
	err := r.db.WithContext(ctx).Where("state = ?", stateStr).First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// DeleteOAuthState deletes an OAuth state
func (r *UserRepository) DeleteOAuthState(ctx context.Context, stateStr string) error {
	return r.db.WithContext(ctx).Where("state = ?", stateStr).Delete(&user.OAuthState{}).Error
}

// CleanupExpiredOAuthStates removes expired OAuth states
func (r *UserRepository) CleanupExpiredOAuthStates(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&user.OAuthState{}).Error
}
