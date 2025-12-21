package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/repository"
)

// SessionService defines the interface for session management
type SessionService interface {
	// CreateSession creates a new session
	CreateSession(ctx context.Context, tenantID, userID uuid.UUID, channelType, channelID string) (*entity.Session, error)

	// GetSession retrieves a session by ID
	GetSession(ctx context.Context, sessionID uuid.UUID) (*entity.Session, error)

	// UpdateSession updates a session
	UpdateSession(ctx context.Context, session *entity.Session) error

	// EndSession ends a session
	EndSession(ctx context.Context, sessionID uuid.UUID) error

	// AddMessage adds a message to a session
	AddMessage(ctx context.Context, sessionID uuid.UUID, message *entity.Message) error

	// GetMessages retrieves messages for a session
	GetMessages(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*entity.Message, error)

	// GetActiveSessionsCount returns the count of active sessions for a tenant
	GetActiveSessionsCount(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// CleanupExpiredSessions removes expired sessions
	CleanupExpiredSessions(ctx context.Context) (int, error)
}

// sessionServiceImpl implements SessionService
type sessionServiceImpl struct {
	sessionRepo repository.SessionRepository
}

// NewSessionService creates a new SessionService
func NewSessionService(sessionRepo repository.SessionRepository) SessionService {
	return &sessionServiceImpl{
		sessionRepo: sessionRepo,
	}
}

// CreateSession creates a new session
func (s *sessionServiceImpl) CreateSession(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	channelType, channelID string,
) (*entity.Session, error) {
	// Validate channel type
	if !isValidChannelType(channelType) {
		return nil, fmt.Errorf("invalid channel type: %s", channelType)
	}

	// Create new session
	session := entity.NewSession(tenantID, userID, channelType, channelID)

	// Save to repository
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (s *sessionServiceImpl) GetSession(ctx context.Context, sessionID uuid.UUID) (*entity.Session, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if expired
	if session.IsExpired() {
		return nil, fmt.Errorf("session has expired")
	}

	return session, nil
}

// UpdateSession updates a session
func (s *sessionServiceImpl) UpdateSession(ctx context.Context, session *entity.Session) error {
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

// EndSession ends a session
func (s *sessionServiceImpl) EndSession(ctx context.Context, sessionID uuid.UUID) error {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to find session: %w", err)
	}

	session.End()

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	return nil
}

// AddMessage adds a message to a session
func (s *sessionServiceImpl) AddMessage(
	ctx context.Context,
	sessionID uuid.UUID,
	message *entity.Message,
) error {
	// Verify session exists and is active
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if !session.IsActive() {
		return fmt.Errorf("session is not active")
	}

	// Add message to repository
	if err := s.sessionRepo.AddMessage(ctx, sessionID, message); err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	// Update session with message
	session.AddMessage(*message)
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// GetMessages retrieves messages for a session
func (s *sessionServiceImpl) GetMessages(
	ctx context.Context,
	sessionID uuid.UUID,
	limit, offset int,
) ([]*entity.Message, error) {
	// Verify session exists
	if _, err := s.GetSession(ctx, sessionID); err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	messages, err := s.sessionRepo.GetMessages(ctx, sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	return messages, nil
}

// GetActiveSessionsCount returns the count of active sessions for a tenant
func (s *sessionServiceImpl) GetActiveSessionsCount(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	sessions, err := s.sessionRepo.FindByTenant(ctx, tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to find sessions: %w", err)
	}

	count := int64(0)
	for _, session := range sessions {
		if session.IsActive() {
			count++
		}
	}

	return count, nil
}

// CleanupExpiredSessions removes expired sessions
func (s *sessionServiceImpl) CleanupExpiredSessions(ctx context.Context) (int, error) {
	expiredSessions, err := s.sessionRepo.FindExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to find expired sessions: %w", err)
	}

	count := 0
	for _, session := range expiredSessions {
		if err := s.sessionRepo.Delete(ctx, session.ID); err != nil {
			// Log error but continue
			continue
		}
		count++
	}

	return count, nil
}

// Helper functions

func isValidChannelType(channelType string) bool {
	validTypes := []string{
		entity.ChannelTypeVoice,
		entity.ChannelTypeText,
		entity.ChannelTypeAPI,
	}

	for _, t := range validTypes {
		if channelType == t {
			return true
		}
	}
	return false
}
