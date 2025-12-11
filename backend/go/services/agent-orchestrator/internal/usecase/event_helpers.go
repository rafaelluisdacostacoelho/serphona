package usecase

import (
	"context"

	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/events"
)

// EventHelpers provides helper methods for publishing events
type EventHelpers struct {
	publisher events.EventPublisher
}

// NewEventHelpers creates a new EventHelpers
func NewEventHelpers(publisher events.EventPublisher) *EventHelpers {
	return &EventHelpers{
		publisher: publisher,
	}
}

// PublishSessionCreated publishes a session created event
func (h *EventHelpers) PublishSessionCreated(ctx context.Context, session *entity.Session) {
	if h.publisher == nil {
		return
	}

	eventData := map[string]interface{}{
		"session_id":   session.ID.String(),
		"user_id":      session.UserID.String(),
		"channel_type": session.ChannelType,
		"channel_id":   session.ChannelID,
	}

	event := events.NewEvent(events.EventTypeSessionCreated, session.TenantID.String(), eventData)
	h.publisher.PublishAsync(ctx, event)
}

// PublishSessionEnded publishes a session ended event
func (h *EventHelpers) PublishSessionEnded(ctx context.Context, session *entity.Session) {
	if h.publisher == nil {
		return
	}

	eventData := map[string]interface{}{
		"session_id": session.ID.String(),
		"user_id":    session.UserID.String(),
	}

	event := events.NewEvent(events.EventTypeSessionEnded, session.TenantID.String(), eventData)
	h.publisher.PublishAsync(ctx, event)
}

// PublishMessageProcessed publishes a message processed event
func (h *EventHelpers) PublishMessageProcessed(ctx context.Context, session *entity.Session, message *entity.Message) {
	if h.publisher == nil {
		return
	}

	toolsExecuted := make([]string, 0)
	for _, tc := range message.ToolCalls {
		toolsExecuted = append(toolsExecuted, tc.ToolName)
	}

	eventData := map[string]interface{}{
		"message_id":     message.ID.String(),
		"session_id":     session.ID.String(),
		"user_id":        session.UserID.String(),
		"agent_name":     message.Metadata.Agent,
		"model":          message.Metadata.Model,
		"tokens":         message.Metadata.Tokens,
		"latency_ms":     message.Metadata.LatencyMS,
		"cost":           message.Metadata.Cost,
		"tools_executed": toolsExecuted,
	}

	event := events.NewEvent(events.EventTypeMessageProcessed, session.TenantID.String(), eventData)
	h.publisher.PublishAsync(ctx, event)
}

// PublishAgentCreated publishes an agent created event
func (h *EventHelpers) PublishAgentCreated(ctx context.Context, agent *entity.Agent) {
	if h.publisher == nil {
		return
	}

	eventData := map[string]interface{}{
		"agent_id":  agent.ID.String(),
		"name":      agent.Name,
		"model":     agent.Model,
		"is_active": agent.IsActive,
	}

	event := events.NewEvent(events.EventTypeAgentCreated, agent.TenantID.String(), eventData)
	h.publisher.PublishAsync(ctx, event)
}

// PublishAgentUpdated publishes an agent updated event
func (h *EventHelpers) PublishAgentUpdated(ctx context.Context, agent *entity.Agent) {
	if h.publisher == nil {
		return
	}

	eventData := map[string]interface{}{
		"agent_id": agent.ID.String(),
		"name":     agent.Name,
	}

	event := events.NewEvent(events.EventTypeAgentUpdated, agent.TenantID.String(), eventData)
	h.publisher.PublishAsync(ctx, event)
}
