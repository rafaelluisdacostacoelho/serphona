package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
)

var errSessionNotFound = errors.New("session not found")

// Mock SessionRepository for testing
type mockSessionRepository struct {
	sessions map[uuid.UUID]*entity.Session
	messages map[uuid.UUID][]*entity.Message
}

func newMockSessionRepository() *mockSessionRepository {
	return &mockSessionRepository{
		sessions: make(map[uuid.UUID]*entity.Session),
		messages: make(map[uuid.UUID][]*entity.Message),
	}
}

func (m *mockSessionRepository) Create(ctx context.Context, session *entity.Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Session, error) {
	session, ok := m.sessions[id]
	if !ok {
		return nil, errSessionNotFound
	}
	return session, nil
}

func (m *mockSessionRepository) CountMessages(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	msgs := m.messages[sessionID]
	return int64(len(msgs)), nil
}

func (m *mockSessionRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*entity.Session, error) {
	var sessions []*entity.Session
	for _, session := range m.sessions {
		if session.TenantID == tenantID {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (m *mockSessionRepository) Update(ctx context.Context, session *entity.Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.sessions, id)
	return nil
}

func (m *mockSessionRepository) AddMessage(ctx context.Context, sessionID uuid.UUID, message *entity.Message) error {
	m.messages[sessionID] = append(m.messages[sessionID], message)
	return nil
}

func (m *mockSessionRepository) GetMessages(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*entity.Message, error) {
	msgs := m.messages[sessionID]
	if msgs == nil {
		return []*entity.Message{}, nil
	}

	// Simple pagination
	start := offset
	end := offset + limit
	if start > len(msgs) {
		return []*entity.Message{}, nil
	}
	if end > len(msgs) {
		end = len(msgs)
	}

	return msgs[start:end], nil
}

func (m *mockSessionRepository) FindExpired(ctx context.Context) ([]*entity.Session, error) {
	var expired []*entity.Session
	now := time.Now()
	for _, session := range m.sessions {
		if session.ExpiresAt.Before(now) {
			expired = append(expired, session)
		}
	}
	return expired, nil
}

func (m *mockSessionRepository) FindActive(ctx context.Context) ([]*entity.Session, error) {
	var active []*entity.Session
	for _, session := range m.sessions {
		if session.Status == entity.SessionStatusActive {
			active = append(active, session)
		}
	}
	return active, nil
}

func (m *mockSessionRepository) FindByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.Session, error) {
	var sessions []*entity.Session
	for _, session := range m.sessions {
		if session.TenantID == tenantID && session.UserID == userID {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

// Tests

func TestSessionService_CreateSession(t *testing.T) {
	repo := newMockSessionRepository()
	service := NewSessionService(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()

	session, err := service.CreateSession(ctx, tenantID, userID, entity.ChannelTypeText, "test-channel")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if session.TenantID != tenantID {
		t.Errorf("Expected tenant ID %v, got %v", tenantID, session.TenantID)
	}

	if session.UserID != userID {
		t.Errorf("Expected user ID %v, got %v", userID, session.UserID)
	}

	if session.ChannelType != entity.ChannelTypeText {
		t.Errorf("Expected channel type %s, got %s", entity.ChannelTypeText, session.ChannelType)
	}
}

func TestSessionService_CreateSession_InvalidChannelType(t *testing.T) {
	repo := newMockSessionRepository()
	service := NewSessionService(repo)
	ctx := context.Background()

	_, err := service.CreateSession(ctx, uuid.New(), uuid.New(), "invalid-type", "test-channel")
	if err == nil {
		t.Fatal("Expected error for invalid channel type, got nil")
	}
}

func TestSessionService_GetSession(t *testing.T) {
	repo := newMockSessionRepository()
	service := NewSessionService(repo)
	ctx := context.Background()

	// Create session
	tenantID := uuid.New()
	userID := uuid.New()
	session, _ := service.CreateSession(ctx, tenantID, userID, entity.ChannelTypeText, "test-channel")

	// Get session
	retrieved, err := service.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("Expected session ID %v, got %v", session.ID, retrieved.ID)
	}
}

func TestSessionService_GetSession_NotFound(t *testing.T) {
	repo := newMockSessionRepository()
	service := NewSessionService(repo)
	ctx := context.Background()

	_, err := service.GetSession(ctx, uuid.New())
	if err == nil {
		t.Fatal("Expected error for non-existent session, got nil")
	}
}

func TestSessionService_EndSession(t *testing.T) {
	repo := newMockSessionRepository()
	service := NewSessionService(repo)
	ctx := context.Background()

	// Create session
	session, _ := service.CreateSession(ctx, uuid.New(), uuid.New(), entity.ChannelTypeText, "test-channel")

	// End session
	err := service.EndSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify session is ended
	retrieved, _ := repo.FindByID(ctx, session.ID)
	if retrieved.Status != entity.SessionStatusEnded {
		t.Errorf("Expected status %s, got %s", entity.SessionStatusEnded, retrieved.Status)
	}
}

func TestSessionService_AddMessage(t *testing.T) {
	repo := newMockSessionRepository()
	service := NewSessionService(repo)
	ctx := context.Background()

	// Create session
	session, _ := service.CreateSession(ctx, uuid.New(), uuid.New(), entity.ChannelTypeText, "test-channel")

	// Add message
	message := entity.NewUserMessage(session.ID, "Hello")
	err := service.AddMessage(ctx, session.ID, message)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Get messages
	messages, _ := service.GetMessages(ctx, session.ID, 10, 0)
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != "Hello" {
		t.Errorf("Expected content 'Hello', got '%s'", messages[0].Content)
	}
}

func TestSessionService_GetActiveSessionsCount(t *testing.T) {
	repo := newMockSessionRepository()
	service := NewSessionService(repo)
	ctx := context.Background()

	tenantID := uuid.New()

	// Create 3 sessions
	service.CreateSession(ctx, tenantID, uuid.New(), entity.ChannelTypeText, "ch1")
	service.CreateSession(ctx, tenantID, uuid.New(), entity.ChannelTypeText, "ch2")
	session3, _ := service.CreateSession(ctx, tenantID, uuid.New(), entity.ChannelTypeText, "ch3")

	// End one session
	service.EndSession(ctx, session3.ID)

	// Get active count
	count, err := service.GetActiveSessionsCount(ctx, tenantID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 active sessions, got %d", count)
	}
}
