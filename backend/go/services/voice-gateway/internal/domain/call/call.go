// Package call contains the call domain model.
package call

import (
	"time"

	"github.com/google/uuid"
)

// Call represents a phone call in the system.
type Call struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	ChannelID      string    `json:"channel_id"` // Asterisk channel ID
	BridgeID       string    `json:"bridge_id"`  // Asterisk bridge ID

	// Call details
	Direction    Direction `json:"direction"`     // inbound, outbound
	CallerNumber string    `json:"caller_number"` // E.164 format
	CalleeNumber string    `json:"callee_number"` // E.164 format

	// State
	State State `json:"state"`

	// Configuration
	TrunkID uuid.UUID  `json:"trunk_id"`
	DIDID   *uuid.UUID `json:"did_id,omitempty"`
	AgentID string     `json:"agent_id"`

	// Providers
	STTProvider string `json:"stt_provider"`
	TTSProvider string `json:"tts_provider"`

	// Timestamps
	CreatedAt  time.Time     `json:"created_at"`
	AnsweredAt *time.Time    `json:"answered_at,omitempty"`
	EndedAt    *time.Time    `json:"ended_at,omitempty"`
	Duration   time.Duration `json:"duration"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Direction represents the call direction.
type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

// State represents the current state of a call.
type State string

const (
	StateRinging     State = "ringing"
	StateAnswered    State = "answered"
	StateActive      State = "active"
	StateHold        State = "hold"
	StateTransferred State = "transferred"
	StateEnded       State = "ended"
	StateError       State = "error"
)

// NewCall creates a new Call.
func NewCall(tenantID uuid.UUID, direction Direction, callerNumber, calleeNumber string) *Call {
	now := time.Now().UTC()
	return &Call{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Direction:    direction,
		CallerNumber: callerNumber,
		CalleeNumber: calleeNumber,
		State:        StateRinging,
		CreatedAt:    now,
		Metadata:     make(map[string]interface{}),
	}
}

// Answer marks the call as answered.
func (c *Call) Answer() {
	now := time.Now().UTC()
	c.AnsweredAt = &now
	c.State = StateAnswered
}

// Activate marks the call as active (conversation started).
func (c *Call) Activate() {
	c.State = StateActive
}

// Hold puts the call on hold.
func (c *Call) Hold() {
	c.State = StateHold
}

// Resume resumes a held call.
func (c *Call) Resume() {
	c.State = StateActive
}

// Transfer marks the call as transferred.
func (c *Call) Transfer() {
	c.State = StateTransferred
}

// End ends the call.
func (c *Call) End() {
	now := time.Now().UTC()
	c.EndedAt = &now
	c.State = StateEnded

	if c.AnsweredAt != nil {
		c.Duration = now.Sub(*c.AnsweredAt)
	}
}

// SetError sets the call to error state.
func (c *Call) SetError() {
	c.State = StateError
	if c.EndedAt == nil {
		now := time.Now().UTC()
		c.EndedAt = &now
	}
}

// IsActive returns true if the call is in an active state.
func (c *Call) IsActive() bool {
	return c.State == StateActive || c.State == StateAnswered || c.State == StateHold
}

// IsEnded returns true if the call has ended.
func (c *Call) IsEnded() bool {
	return c.State == StateEnded || c.State == StateError
}
