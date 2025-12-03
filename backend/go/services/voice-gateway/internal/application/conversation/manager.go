// Package conversation provides conversation management.
package conversation

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"voice-gateway/internal/adapter/events"
)

// Manager manages active conversations and their state.
type Manager struct {
	conversations  map[uuid.UUID]*Conversation
	mu             sync.RWMutex
	logger         *zap.Logger
	eventPublisher *events.Publisher
}

// NewManager creates a new conversation manager.
func NewManager(eventPublisher *events.Publisher, logger *zap.Logger) *Manager {
	return &Manager{
		conversations:  make(map[uuid.UUID]*Conversation),
		logger:         logger,
		eventPublisher: eventPublisher,
	}
}

// Conversation represents an active conversation.
type Conversation struct {
	ID        uuid.UUID
	CallID    uuid.UUID
	TenantID  uuid.UUID
	AgentID   string
	Context   map[string]interface{}
	TurnCount int
	MaxTurns  int
	Active    bool
}

// CreateConversation creates a new conversation.
func (m *Manager) CreateConversation(ctx context.Context, callID, tenantID uuid.UUID, agentID string, maxTurns int) (*Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conv := &Conversation{
		ID:        uuid.New(),
		CallID:    callID,
		TenantID:  tenantID,
		AgentID:   agentID,
		Context:   make(map[string]interface{}),
		TurnCount: 0,
		MaxTurns:  maxTurns,
		Active:    true,
	}

	m.conversations[conv.ID] = conv

	m.logger.Info("conversation created",
		zap.String("conversation_id", conv.ID.String()),
		zap.String("call_id", callID.String()),
		zap.String("agent_id", agentID),
	)

	return conv, nil
}

// GetConversation retrieves a conversation by ID.
func (m *Manager) GetConversation(conversationID uuid.UUID) (*Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conv, ok := m.conversations[conversationID]
	if !ok {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	return conv, nil
}

// AddTurn increments the turn counter for a conversation.
func (m *Manager) AddTurn(conversationID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conv, ok := m.conversations[conversationID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}

	conv.TurnCount++

	// Check if max turns reached
	if conv.MaxTurns > 0 && conv.TurnCount >= conv.MaxTurns {
		conv.Active = false
		m.logger.Warn("conversation reached max turns",
			zap.String("conversation_id", conversationID.String()),
			zap.Int("turns", conv.TurnCount),
		)
	}

	return nil
}

// SetContext sets a context value for the conversation.
func (m *Manager) SetContext(conversationID uuid.UUID, key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conv, ok := m.conversations[conversationID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}

	conv.Context[key] = value
	return nil
}

// GetContext retrieves a context value from the conversation.
func (m *Manager) GetContext(conversationID uuid.UUID, key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conv, ok := m.conversations[conversationID]
	if !ok {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	value, ok := conv.Context[key]
	if !ok {
		return nil, fmt.Errorf("context key not found: %s", key)
	}

	return value, nil
}

// EndConversation marks a conversation as ended and removes it.
func (m *Manager) EndConversation(conversationID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conv, ok := m.conversations[conversationID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}

	conv.Active = false
	delete(m.conversations, conversationID)

	m.logger.Info("conversation ended",
		zap.String("conversation_id", conversationID.String()),
		zap.Int("total_turns", conv.TurnCount),
	)

	return nil
}

// IsActive checks if a conversation is still active.
func (m *Manager) IsActive(conversationID uuid.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conv, ok := m.conversations[conversationID]
	if !ok {
		return false
	}

	return conv.Active
}

// ListActiveConversations returns all active conversations.
func (m *Manager) ListActiveConversations() []*Conversation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conversations := make([]*Conversation, 0, len(m.conversations))
	for _, conv := range m.conversations {
		if conv.Active {
			conversations = append(conversations, conv)
		}
	}

	return conversations
}
