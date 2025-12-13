// Package conversation contains the conversation domain model.
package conversation

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Conversation represents an AI conversation within a call.
type Conversation struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	CallID    uuid.UUID `json:"call_id"`
	AgentID   string    `json:"agent_id"`
	SessionID string    `json:"session_id"` // External session ID (e.g., agent-orchestrator)

	// State
	State  State  `json:"state"`
	Turns  []Turn `json:"turns"`
	Active bool   `json:"active"`

	// Context
	Context map[string]interface{} `json:"context,omitempty"`

	// Timestamps
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// State represents the conversation state.
type State string

const (
	StateInitialized State = "initialized"
	StateActive      State = "active"
	StatePaused      State = "paused"
	StateCompleted   State = "completed"
	StateError       State = "error"
)

// Turn represents a conversation turn (user speaks, agent responds).
type Turn struct {
	ID        uuid.UUID `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Speaker   Speaker   `json:"speaker"` // user or agent
	Content   string    `json:"content"`
	Duration  int       `json:"duration_ms,omitempty"` // Audio duration in milliseconds

	// Processing metrics
	STTLatency int `json:"stt_latency_ms,omitempty"`
	LLMLatency int `json:"llm_latency_ms,omitempty"`
	TTSLatency int `json:"tts_latency_ms,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Speaker represents who is speaking.
type Speaker string

const (
	SpeakerUser  Speaker = "user"
	SpeakerAgent Speaker = "agent"
)

// NewConversation creates a new Conversation.
func NewConversation(tenantID, callID uuid.UUID, agentID string) *Conversation {
	return &Conversation{
		ID:        uuid.New(),
		TenantID:  tenantID,
		CallID:    callID,
		AgentID:   agentID,
		State:     StateInitialized,
		Turns:     make([]Turn, 0),
		Active:    false,
		Context:   make(map[string]interface{}),
		StartedAt: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}
}

// Start activates the conversation.
func (c *Conversation) Start() {
	c.State = StateActive
	c.Active = true
}

// Pause pauses the conversation temporarily.
func (c *Conversation) Pause() {
	if c.State == StateActive {
		c.State = StatePaused
	}
}

// Resume resumes a paused conversation.
func (c *Conversation) Resume() error {
	if c.State != StatePaused {
		return errors.New("can only resume paused conversations")
	}
	c.State = StateActive
	return nil
}

// Complete marks the conversation as completed.
func (c *Conversation) Complete() {
	now := time.Now().UTC()
	c.EndedAt = &now
	c.State = StateCompleted
	c.Active = false
}

// SetError sets the conversation to error state.
func (c *Conversation) SetError() {
	now := time.Now().UTC()
	c.EndedAt = &now
	c.State = StateError
	c.Active = false
}

// AddUserTurn adds a user's speech turn.
func (c *Conversation) AddUserTurn(content string, durationMs int) *Turn {
	turn := &Turn{
		ID:        uuid.New(),
		Timestamp: time.Now().UTC(),
		Speaker:   SpeakerUser,
		Content:   content,
		Duration:  durationMs,
		Metadata:  make(map[string]interface{}),
	}
	c.Turns = append(c.Turns, *turn)
	return turn
}

// AddAgentTurn adds an agent's response turn.
func (c *Conversation) AddAgentTurn(content string, durationMs int) *Turn {
	turn := &Turn{
		ID:        uuid.New(),
		Timestamp: time.Now().UTC(),
		Speaker:   SpeakerAgent,
		Content:   content,
		Duration:  durationMs,
		Metadata:  make(map[string]interface{}),
	}
	c.Turns = append(c.Turns, *turn)
	return turn
}

// GetLastTurn returns the last turn in the conversation.
func (c *Conversation) GetLastTurn() *Turn {
	if len(c.Turns) == 0 {
		return nil
	}
	return &c.Turns[len(c.Turns)-1]
}

// GetTurnCount returns the total number of turns.
func (c *Conversation) GetTurnCount() int {
	return len(c.Turns)
}

// GetUserTurnCount returns the number of user turns.
func (c *Conversation) GetUserTurnCount() int {
	count := 0
	for _, turn := range c.Turns {
		if turn.Speaker == SpeakerUser {
			count++
		}
	}
	return count
}

// GetAgentTurnCount returns the number of agent turns.
func (c *Conversation) GetAgentTurnCount() int {
	count := 0
	for _, turn := range c.Turns {
		if turn.Speaker == SpeakerAgent {
			count++
		}
	}
	return count
}

// GetDuration returns the conversation duration.
func (c *Conversation) GetDuration() time.Duration {
	if c.EndedAt != nil {
		return c.EndedAt.Sub(c.StartedAt)
	}
	return time.Since(c.StartedAt)
}

// IsActive returns true if the conversation is active.
func (c *Conversation) IsActive() bool {
	return c.Active && c.State == StateActive
}

// CanAddTurn returns true if new turns can be added.
func (c *Conversation) CanAddTurn() bool {
	return c.State == StateActive || c.State == StateInitialized
}

// SetContext sets a context value.
func (c *Conversation) SetContext(key string, value interface{}) {
	if c.Context == nil {
		c.Context = make(map[string]interface{})
	}
	c.Context[key] = value
}

// GetContext gets a context value.
func (c *Conversation) GetContext(key string) (interface{}, bool) {
	if c.Context == nil {
		return nil, false
	}
	val, ok := c.Context[key]
	return val, ok
}

// UpdateTurnMetrics updates processing metrics for a turn.
func (c *Conversation) UpdateTurnMetrics(turnID uuid.UUID, sttLatency, llmLatency, ttsLatency int) error {
	for i := range c.Turns {
		if c.Turns[i].ID == turnID {
			c.Turns[i].STTLatency = sttLatency
			c.Turns[i].LLMLatency = llmLatency
			c.Turns[i].TTSLatency = ttsLatency
			return nil
		}
	}
	return errors.New("turn not found")
}

// GetAverageSTTLatency calculates average STT latency across all turns.
func (c *Conversation) GetAverageSTTLatency() float64 {
	if len(c.Turns) == 0 {
		return 0
	}

	total := 0
	count := 0
	for _, turn := range c.Turns {
		if turn.STTLatency > 0 {
			total += turn.STTLatency
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return float64(total) / float64(count)
}

// GetAverageLLMLatency calculates average LLM latency across all turns.
func (c *Conversation) GetAverageLLMLatency() float64 {
	if len(c.Turns) == 0 {
		return 0
	}

	total := 0
	count := 0
	for _, turn := range c.Turns {
		if turn.LLMLatency > 0 {
			total += turn.LLMLatency
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return float64(total) / float64(count)
}

// GetAverageTTSLatency calculates average TTS latency across all turns.
func (c *Conversation) GetAverageTTSLatency() float64 {
	if len(c.Turns) == 0 {
		return 0
	}

	total := 0
	count := 0
	for _, turn := range c.Turns {
		if turn.TTSLatency > 0 {
			total += turn.TTSLatency
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return float64(total) / float64(count)
}
