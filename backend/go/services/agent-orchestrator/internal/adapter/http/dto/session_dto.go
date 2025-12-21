package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
)

// CreateSessionRequest represents a request to create a session
type CreateSessionRequest struct {
	TenantID    uuid.UUID `json:"tenant_id" binding:"required"`
	UserID      uuid.UUID `json:"user_id" binding:"required"`
	ChannelType string    `json:"channel_type" binding:"required,oneof=voice text api"`
	ChannelID   string    `json:"channel_id" binding:"required"`
}

// SessionResponse represents a session response
type SessionResponse struct {
	ID          uuid.UUID              `json:"id"`
	TenantID    uuid.UUID              `json:"tenant_id"`
	UserID      uuid.UUID              `json:"user_id"`
	ChannelType string                 `json:"channel_type"`
	ChannelID   string                 `json:"channel_id"`
	Status      string                 `json:"status"`
	Context     SessionContextResponse `json:"context"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
}

// SessionContextResponse represents session context
type SessionContextResponse struct {
	MessageCount    int                    `json:"message_count"`
	Variables       map[string]interface{} `json:"variables"`
	CurrentAgent    string                 `json:"current_agent"`
	DelegationChain []string               `json:"delegation_chain"`
}

// ToSessionResponse converts entity to DTO
func ToSessionResponse(session *entity.Session) *SessionResponse {
	return &SessionResponse{
		ID:          session.ID,
		TenantID:    session.TenantID,
		UserID:      session.UserID,
		ChannelType: session.ChannelType,
		ChannelID:   session.ChannelID,
		Status:      session.Status,
		Context: SessionContextResponse{
			MessageCount:    len(session.Context.Messages),
			Variables:       session.Context.Variables,
			CurrentAgent:    session.Context.CurrentAgent,
			DelegationChain: session.Context.DelegationChain,
		},
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
		ExpiresAt: session.ExpiresAt,
	}
}
