package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
)

// SessionRepository defines the interface for session persistence
type SessionRepository interface {
	// Create creates a new session
	Create(ctx context.Context, session *entity.Session) error

	// FindByID finds a session by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Session, error)

	// Update updates a session
	Update(ctx context.Context, session *entity.Session) error

	// Delete deletes a session
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByTenant finds all sessions for a tenant
	FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*entity.Session, error)

	// FindByUser finds all sessions for a user
	FindByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.Session, error)

	// FindActive finds all active sessions
	FindActive(ctx context.Context) ([]*entity.Session, error)

	// FindExpired finds all expired sessions
	FindExpired(ctx context.Context) ([]*entity.Session, error)

	// AddMessage adds a message to a session
	AddMessage(ctx context.Context, sessionID uuid.UUID, message *entity.Message) error

	// GetMessages retrieves messages for a session
	GetMessages(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*entity.Message, error)

	// CountMessages counts messages in a session
	CountMessages(ctx context.Context, sessionID uuid.UUID) (int64, error)
}
