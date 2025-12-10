package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Message represents a message in a conversation
type Message struct {
	ID        uuid.UUID       `json:"id"`
	SessionID uuid.UUID       `json:"session_id"`
	Role      string          `json:"role"` // user, assistant, system, tool
	Content   string          `json:"content"`
	ToolCalls []ToolCall      `json:"tool_calls,omitempty"`
	Metadata  MessageMetadata `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
}

// ToolCall represents a function/tool call from the LLM
type ToolCall struct {
	ID        string          `json:"id"`
	ToolName  string          `json:"tool_name"`
	Arguments json.RawMessage `json:"arguments"`
	Result    json.RawMessage `json:"result,omitempty"`
	Status    string          `json:"status"` // pending, success, error
	Error     string          `json:"error,omitempty"`
}

// MessageMetadata contains metadata about message generation
type MessageMetadata struct {
	Model        string  `json:"model"`
	Tokens       int     `json:"tokens"`
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
	LatencyMS    int64   `json:"latency_ms"`
	Agent        string  `json:"agent"`
	Cost         float64 `json:"cost"`
}

// Message role constants
const (
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
	MessageRoleSystem    = "system"
	MessageRoleTool      = "tool"
)

// Tool call status constants
const (
	ToolCallStatusPending = "pending"
	ToolCallStatusSuccess = "success"
	ToolCallStatusError   = "error"
)

// NewMessage creates a new message
func NewMessage(sessionID uuid.UUID, role, content string) *Message {
	return &Message{
		ID:        uuid.New(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		ToolCalls: []ToolCall{},
		Metadata:  MessageMetadata{},
		CreatedAt: time.Now(),
	}
}

// NewUserMessage creates a new user message
func NewUserMessage(sessionID uuid.UUID, content string) *Message {
	return NewMessage(sessionID, MessageRoleUser, content)
}

// NewAssistantMessage creates a new assistant message
func NewAssistantMessage(sessionID uuid.UUID, content string) *Message {
	msg := NewMessage(sessionID, MessageRoleAssistant, content)
	return msg
}

// NewSystemMessage creates a new system message
func NewSystemMessage(sessionID uuid.UUID, content string) *Message {
	return NewMessage(sessionID, MessageRoleSystem, content)
}

// AddToolCall adds a tool call to the message
func (m *Message) AddToolCall(toolCall ToolCall) {
	m.ToolCalls = append(m.ToolCalls, toolCall)
}

// HasToolCalls checks if the message has any tool calls
func (m *Message) HasToolCalls() bool {
	return len(m.ToolCalls) > 0
}

// SetMetadata sets the message metadata
func (m *Message) SetMetadata(metadata MessageMetadata) {
	m.Metadata = metadata
}

// CalculateCost calculates the cost based on tokens and model
func (m *Message) CalculateCost() {
	// Cost calculation based on model
	// These are approximate rates (as of 2024)
	switch m.Metadata.Model {
	case "gpt-4-turbo", "gpt-4-1106-preview":
		// $0.01 per 1K input tokens, $0.03 per 1K output tokens
		inputCost := float64(m.Metadata.InputTokens) / 1000.0 * 0.01
		outputCost := float64(m.Metadata.OutputTokens) / 1000.0 * 0.03
		m.Metadata.Cost = inputCost + outputCost
	case "gpt-4":
		// $0.03 per 1K input tokens, $0.06 per 1K output tokens
		inputCost := float64(m.Metadata.InputTokens) / 1000.0 * 0.03
		outputCost := float64(m.Metadata.OutputTokens) / 1000.0 * 0.06
		m.Metadata.Cost = inputCost + outputCost
	case "gpt-3.5-turbo":
		// $0.001 per 1K input tokens, $0.002 per 1K output tokens
		inputCost := float64(m.Metadata.InputTokens) / 1000.0 * 0.001
		outputCost := float64(m.Metadata.OutputTokens) / 1000.0 * 0.002
		m.Metadata.Cost = inputCost + outputCost
	case "claude-3-opus-20240229":
		// $0.015 per 1K input tokens, $0.075 per 1K output tokens
		inputCost := float64(m.Metadata.InputTokens) / 1000.0 * 0.015
		outputCost := float64(m.Metadata.OutputTokens) / 1000.0 * 0.075
		m.Metadata.Cost = inputCost + outputCost
	case "claude-3-sonnet-20240229":
		// $0.003 per 1K input tokens, $0.015 per 1K output tokens
		inputCost := float64(m.Metadata.InputTokens) / 1000.0 * 0.003
		outputCost := float64(m.Metadata.OutputTokens) / 1000.0 * 0.015
		m.Metadata.Cost = inputCost + outputCost
	default:
		// Default to gpt-3.5-turbo pricing
		inputCost := float64(m.Metadata.InputTokens) / 1000.0 * 0.001
		outputCost := float64(m.Metadata.OutputTokens) / 1000.0 * 0.002
		m.Metadata.Cost = inputCost + outputCost
	}
}

// ToJSON converts the message to JSON
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}
