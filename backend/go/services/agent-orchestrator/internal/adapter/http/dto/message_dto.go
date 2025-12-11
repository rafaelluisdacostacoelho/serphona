package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/usecase"
)

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	Content string    `json:"content" binding:"required"`
	UserID  uuid.UUID `json:"user_id" binding:"required"`
}

// MessageResponse represents a message response
type MessageResponse struct {
	ID        uuid.UUID       `json:"id"`
	SessionID uuid.UUID       `json:"session_id"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	ToolCalls []ToolCallDTO   `json:"tool_calls,omitempty"`
	Metadata  MessageMetadata `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
}

// ToolCallDTO represents a tool call
type ToolCallDTO struct {
	ID        string      `json:"id"`
	ToolName  string      `json:"tool_name"`
	Arguments interface{} `json:"arguments"`
	Result    interface{} `json:"result,omitempty"`
	Status    string      `json:"status"`
	Error     string      `json:"error,omitempty"`
}

// MessageMetadata represents message metadata
type MessageMetadata struct {
	Model        string  `json:"model"`
	Tokens       int     `json:"tokens"`
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
	LatencyMS    int64   `json:"latency_ms"`
	Agent        string  `json:"agent"`
	Cost         float64 `json:"cost"`
}

// ToMessageResponse converts entity to DTO
func ToMessageResponse(msg *entity.Message) *MessageResponse {
	toolCalls := make([]ToolCallDTO, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		toolCalls = append(toolCalls, ToolCallDTO{
			ID:        tc.ID,
			ToolName:  tc.ToolName,
			Arguments: tc.Arguments,
			Result:    tc.Result,
			Status:    tc.Status,
			Error:     tc.Error,
		})
	}

	return &MessageResponse{
		ID:        msg.ID,
		SessionID: msg.SessionID,
		Role:      msg.Role,
		Content:   msg.Content,
		ToolCalls: toolCalls,
		Metadata: MessageMetadata{
			Model:        msg.Metadata.Model,
			Tokens:       msg.Metadata.Tokens,
			InputTokens:  msg.Metadata.InputTokens,
			OutputTokens: msg.Metadata.OutputTokens,
			LatencyMS:    msg.Metadata.LatencyMS,
			Agent:        msg.Metadata.Agent,
			Cost:         msg.Metadata.Cost,
		},
		CreatedAt: msg.CreatedAt,
	}
}

// ToProcessMessageResponse converts usecase response to DTO
func ToProcessMessageResponse(resp *usecase.ProcessMessageResponse) *MessageResponse {
	toolCalls := make([]ToolCallDTO, 0, len(resp.ToolCalls))
	for _, tc := range resp.ToolCalls {
		toolCalls = append(toolCalls, ToolCallDTO{
			ID:        tc.ID,
			ToolName:  tc.ToolName,
			Arguments: tc.Arguments,
			Result:    tc.Result,
			Status:    tc.Status,
			Error:     tc.Error,
		})
	}

	return &MessageResponse{
		ID:        resp.MessageID,
		Role:      entity.MessageRoleAssistant,
		Content:   resp.Content,
		ToolCalls: toolCalls,
		Metadata: MessageMetadata{
			Model:        resp.Metadata.Model,
			Tokens:       resp.Metadata.Tokens,
			InputTokens:  resp.Metadata.InputTokens,
			OutputTokens: resp.Metadata.OutputTokens,
			LatencyMS:    resp.Metadata.LatencyMS,
			Agent:        resp.Metadata.Agent,
			Cost:         resp.Metadata.Cost,
		},
	}
}
