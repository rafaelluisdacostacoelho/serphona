package events

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of event
type EventType string

// Event types
const (
	EventTypeSessionCreated   EventType = "session.created"
	EventTypeSessionEnded     EventType = "session.ended"
	EventTypeMessageProcessed EventType = "message.processed"
	EventTypeAgentCreated     EventType = "agent.created"
	EventTypeAgentUpdated     EventType = "agent.updated"
	EventTypeAgentActivated   EventType = "agent.activated"
	EventTypeAgentDeactivated EventType = "agent.deactivated"
	EventTypeToolExecuted     EventType = "tool.executed"
)

// Event represents a base event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	TenantID  string                 `json:"tenant_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewEvent creates a new event
func NewEvent(eventType EventType, tenantID string, data map[string]interface{}) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		TenantID:  tenantID,
		Data:      data,
		Timestamp: time.Now().UTC(),
	}
}

// SessionCreatedEvent represents a session created event
type SessionCreatedEvent struct {
	SessionID   string `json:"session_id"`
	TenantID    string `json:"tenant_id"`
	UserID      string `json:"user_id"`
	ChannelType string `json:"channel_type"`
	ChannelID   string `json:"channel_id"`
}

// SessionEndedEvent represents a session ended event
type SessionEndedEvent struct {
	SessionID string `json:"session_id"`
	TenantID  string `json:"tenant_id"`
	UserID    string `json:"user_id"`
	Duration  int64  `json:"duration_seconds"`
}

// MessageProcessedEvent represents a message processed event
type MessageProcessedEvent struct {
	MessageID     string   `json:"message_id"`
	SessionID     string   `json:"session_id"`
	TenantID      string   `json:"tenant_id"`
	UserID        string   `json:"user_id"`
	AgentName     string   `json:"agent_name"`
	Model         string   `json:"model"`
	Tokens        int      `json:"tokens"`
	LatencyMS     int64    `json:"latency_ms"`
	Cost          float64  `json:"cost"`
	ToolsExecuted []string `json:"tools_executed,omitempty"`
}

// AgentCreatedEvent represents an agent created event
type AgentCreatedEvent struct {
	AgentID  string `json:"agent_id"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	Model    string `json:"model"`
	IsActive bool   `json:"is_active"`
}

// AgentUpdatedEvent represents an agent updated event
type AgentUpdatedEvent struct {
	AgentID  string `json:"agent_id"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
}

// AgentActivatedEvent represents an agent activated event
type AgentActivatedEvent struct {
	AgentID  string `json:"agent_id"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
}

// AgentDeactivatedEvent represents an agent deactivated event
type AgentDeactivatedEvent struct {
	AgentID  string `json:"agent_id"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
}

// ToolExecutedEvent represents a tool executed event
type ToolExecutedEvent struct {
	ExecutionID string `json:"execution_id"`
	SessionID   string `json:"session_id"`
	TenantID    string `json:"tenant_id"`
	ToolName    string `json:"tool_name"`
	Status      string `json:"status"`
	Duration    int64  `json:"duration_ms"`
}
