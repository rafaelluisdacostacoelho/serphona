package events

import (
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/topics"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/types"
)

func TestAgentUpdatedEventBind(t *testing.T) {
	payload := AgentUpdatedEvent{
		AgentID:  "agent-123",
		TenantID: "tenant-456",
		Changes: map[string]string{
			"model":   "gpt-5",
			"version": "1.2.0",
		},
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
		UpdatedBy: "user-999",
	}

	event := NewEvent(topics.AgentUpdated, "agent-orchestrator", payload)
	decoded, err := types.Bind[AgentUpdatedEvent](event)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.AgentID != payload.AgentID {
		t.Fatalf("AgentID mismatch: got %s want %s", decoded.AgentID, payload.AgentID)
	}
	if decoded.TenantID != payload.TenantID {
		t.Fatalf("TenantID mismatch: got %s want %s", decoded.TenantID, payload.TenantID)
	}
	if len(decoded.Changes) != len(payload.Changes) {
		t.Fatalf("Changes length mismatch: got %d want %d", len(decoded.Changes), len(payload.Changes))
	}
	if decoded.Changes["model"] != payload.Changes["model"] {
		t.Fatalf("Changes[model] mismatch: got %s want %s", decoded.Changes["model"], payload.Changes["model"])
	}
	if !decoded.UpdatedAt.Equal(payload.UpdatedAt) {
		t.Fatalf("UpdatedAt mismatch: got %s want %s", decoded.UpdatedAt, payload.UpdatedAt)
	}
	if decoded.UpdatedBy != payload.UpdatedBy {
		t.Fatalf("UpdatedBy mismatch: got %s want %s", decoded.UpdatedBy, payload.UpdatedBy)
	}
}

func TestAgentStoppedEventBind(t *testing.T) {
	payload := AgentStoppedEvent{
		AgentID:   "agent-123",
		TenantID:  "tenant-456",
		StoppedAt: time.Now().UTC().Truncate(time.Second),
		Reason:    "deployment_update",
		Code:      "ROLLING_DEPLOY",
	}

	event := NewEvent(topics.AgentStopped, "agent-orchestrator", payload)
	decoded, err := types.Bind[AgentStoppedEvent](event)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.AgentID != payload.AgentID {
		t.Fatalf("AgentID mismatch: got %s want %s", decoded.AgentID, payload.AgentID)
	}
	if decoded.TenantID != payload.TenantID {
		t.Fatalf("TenantID mismatch: got %s want %s", decoded.TenantID, payload.TenantID)
	}
	if !decoded.StoppedAt.Equal(payload.StoppedAt) {
		t.Fatalf("StoppedAt mismatch: got %s want %s", decoded.StoppedAt, payload.StoppedAt)
	}
	if decoded.Reason != payload.Reason {
		t.Fatalf("Reason mismatch: got %s want %s", decoded.Reason, payload.Reason)
	}
	if decoded.Code != payload.Code {
		t.Fatalf("Code mismatch: got %s want %s", decoded.Code, payload.Code)
	}
}

func TestMessageReceivedEventBind(t *testing.T) {
	payload := MessageReceivedEvent{
		MessageID:      "msg-123",
		ConversationID: "conv-456",
		AgentID:        "agent-789",
		TenantID:       "tenant-000",
		Content:        "Hello, I need help",
		ContentType:    "text/plain",
		Channel:        "voice",
		Source:         "customer",
		ReceivedAt:     time.Now().UTC().Truncate(time.Second),
		Sender:         "user@example.com",
	}

	event := NewEvent(topics.MessageReceived, "voice-gateway", payload)
	decoded, err := types.Bind[MessageReceivedEvent](event)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.MessageID != payload.MessageID {
		t.Fatalf("MessageID mismatch: got %s want %s", decoded.MessageID, payload.MessageID)
	}
	if decoded.ConversationID != payload.ConversationID {
		t.Fatalf("ConversationID mismatch: got %s want %s", decoded.ConversationID, payload.ConversationID)
	}
	if decoded.AgentID != payload.AgentID {
		t.Fatalf("AgentID mismatch: got %s want %s", decoded.AgentID, payload.AgentID)
	}
	if decoded.TenantID != payload.TenantID {
		t.Fatalf("TenantID mismatch: got %s want %s", decoded.TenantID, payload.TenantID)
	}
	if decoded.Content != payload.Content {
		t.Fatalf("Content mismatch: got %s want %s", decoded.Content, payload.Content)
	}
	if decoded.ContentType != payload.ContentType {
		t.Fatalf("ContentType mismatch: got %s want %s", decoded.ContentType, payload.ContentType)
	}
	if decoded.Channel != payload.Channel {
		t.Fatalf("Channel mismatch: got %s want %s", decoded.Channel, payload.Channel)
	}
	if decoded.Source != payload.Source {
		t.Fatalf("Source mismatch: got %s want %s", decoded.Source, payload.Source)
	}
	if !decoded.ReceivedAt.Equal(payload.ReceivedAt) {
		t.Fatalf("ReceivedAt mismatch: got %s want %s", decoded.ReceivedAt, payload.ReceivedAt)
	}
	if decoded.Sender != payload.Sender {
		t.Fatalf("Sender mismatch: got %s want %s", decoded.Sender, payload.Sender)
	}
}
