package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Session represents a conversation session
type Session struct {
	ID          uuid.UUID      `json:"id"`
	TenantID    uuid.UUID      `json:"tenant_id"`
	UserID      uuid.UUID      `json:"user_id"`
	ChannelType string         `json:"channel_type"` // voice, text, api
	ChannelID   string         `json:"channel_id"`   // phone number, chat id, etc
	Status      string         `json:"status"`       // active, ended, timeout
	Context     SessionContext `json:"context"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	ExpiresAt   time.Time      `json:"expires_at"`
}

// SessionContext holds the conversation context
type SessionContext struct {
	Messages        []Message              `json:"messages"`
	Variables       map[string]interface{} `json:"variables"`
	CurrentAgent    string                 `json:"current_agent"`
	DelegationChain []string               `json:"delegation_chain"`
}

// Session status constants
const (
	SessionStatusActive  = "active"
	SessionStatusEnded   = "ended"
	SessionStatusTimeout = "timeout"
)

// Channel type constants
const (
	ChannelTypeVoice = "voice"
	ChannelTypeText  = "text"
	ChannelTypeAPI   = "api"
)

// NewSession creates a new session
func NewSession(tenantID, userID uuid.UUID, channelType, channelID string) *Session {
	now := time.Now()
	return &Session{
		ID:          uuid.New(),
		TenantID:    tenantID,
		UserID:      userID,
		ChannelType: channelType,
		ChannelID:   channelID,
		Status:      SessionStatusActive,
		Context: SessionContext{
			Messages:        []Message{},
			Variables:       make(map[string]interface{}),
			DelegationChain: []string{},
		},
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour), // Default 24h TTL
	}
}

// AddMessage adds a message to the session context
func (s *Session) AddMessage(msg Message) {
	s.Context.Messages = append(s.Context.Messages, msg)
	s.UpdatedAt = time.Now()
}

// SetVariable sets a context variable
func (s *Session) SetVariable(key string, value interface{}) {
	s.Context.Variables[key] = value
	s.UpdatedAt = time.Now()
}

// GetVariable gets a context variable
func (s *Session) GetVariable(key string) (interface{}, bool) {
	val, ok := s.Context.Variables[key]
	return val, ok
}

// SetCurrentAgent sets the current agent
func (s *Session) SetCurrentAgent(agentName string) {
	s.Context.CurrentAgent = agentName
	s.UpdatedAt = time.Now()
}

// AddToDelegationChain adds an agent to the delegation chain
func (s *Session) AddToDelegationChain(agentName string) {
	s.Context.DelegationChain = append(s.Context.DelegationChain, agentName)
	s.UpdatedAt = time.Now()
}

// End marks the session as ended
func (s *Session) End() {
	s.Status = SessionStatusEnded
	s.UpdatedAt = time.Now()
}

// IsActive checks if the session is active
func (s *Session) IsActive() bool {
	return s.Status == SessionStatusActive && time.Now().Before(s.ExpiresAt)
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// GetMessageCount returns the number of messages in the session
func (s *Session) GetMessageCount() int {
	return len(s.Context.Messages)
}

// GetLastMessage returns the last message in the session
func (s *Session) GetLastMessage() *Message {
	if len(s.Context.Messages) == 0 {
		return nil
	}
	return &s.Context.Messages[len(s.Context.Messages)-1]
}

// ToJSON converts the session to JSON
func (s *Session) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// FromJSON creates a session from JSON
func FromJSON(data []byte) (*Session, error) {
	var session Session
	err := json.Unmarshal(data, &session)
	return &session, err
}
