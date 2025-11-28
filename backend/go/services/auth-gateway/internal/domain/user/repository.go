package user

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for user data access
type Repository interface {
	// User operations
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByProvider(ctx context.Context, provider, providerID string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Session operations
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, refreshToken string) (*Session, error)
	RevokeSession(ctx context.Context, refreshToken string) error
	RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error
	CleanupExpiredSessions(ctx context.Context) error

	// OAuth operations
	CreateOAuthState(ctx context.Context, state *OAuthState) error
	GetOAuthState(ctx context.Context, stateStr string) (*OAuthState, error)
	DeleteOAuthState(ctx context.Context, stateStr string) error
	CleanupExpiredOAuthStates(ctx context.Context) error
}
