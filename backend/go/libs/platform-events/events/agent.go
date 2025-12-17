package events

import "time"

// AgentCreatedEvent announces the creation of a conversational agent instance.
type AgentCreatedEvent struct {
	AgentID      string    `json:"agent_id"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	Channel      string    `json:"channel"`
	Model        string    `json:"model"`
	Version      string    `json:"version,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	CreatedBy    string    `json:"created_by,omitempty"`
	Description  string    `json:"description,omitempty"`
	Capabilities []string  `json:"capabilities,omitempty"`
}

// AgentUpdatedEvent tracks mutable changes applied to an agent configuration.
type AgentUpdatedEvent struct {
	AgentID   string            `json:"agent_id"`
	TenantID  string            `json:"tenant_id"`
	Changes   map[string]string `json:"changes,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
	UpdatedBy string            `json:"updated_by,omitempty"`
}

// AgentDeletedEvent signals that an agent has been removed from service.
type AgentDeletedEvent struct {
	AgentID   string    `json:"agent_id"`
	TenantID  string    `json:"tenant_id"`
	DeletedAt time.Time `json:"deleted_at"`
	DeletedBy string    `json:"deleted_by,omitempty"`
	Reason    string    `json:"reason,omitempty"`
}

// AgentDeployedEvent captures deployment metadata for an agent rollout.
type AgentDeployedEvent struct {
	AgentID     string    `json:"agent_id"`
	TenantID    string    `json:"tenant_id"`
	Environment string    `json:"environment"`
	Version     string    `json:"version"`
	DeployedAt  time.Time `json:"deployed_at"`
	DeployedBy  string    `json:"deployed_by,omitempty"`
	ArtifactID  string    `json:"artifact_id,omitempty"`
}

// AgentStartedEvent informs systems that an agent runtime boot completed.
type AgentStartedEvent struct {
	AgentID   string    `json:"agent_id"`
	TenantID  string    `json:"tenant_id"`
	StartedAt time.Time `json:"started_at"`
	Node      string    `json:"node,omitempty"`
	Cluster   string    `json:"cluster,omitempty"`
}

// AgentStoppedEvent highlights an agent shutdown and the reason behind it.
type AgentStoppedEvent struct {
	AgentID   string    `json:"agent_id"`
	TenantID  string    `json:"tenant_id"`
	StoppedAt time.Time `json:"stopped_at"`
	Reason    string    `json:"reason,omitempty"`
	Code      string    `json:"code,omitempty"`
}

// ConversationStartedEvent tracks when an agent session begins with a participant.
type ConversationStartedEvent struct {
	ConversationID string    `json:"conversation_id"`
	AgentID        string    `json:"agent_id"`
	TenantID       string    `json:"tenant_id"`
	Channel        string    `json:"channel"`
	StartedAt      time.Time `json:"started_at"`
	UserID         string    `json:"user_id,omitempty"`
	CorrelationID  string    `json:"correlation_id,omitempty"`
}

// ConversationEndedEvent documents the end of an agent conversation.
type ConversationEndedEvent struct {
	ConversationID string        `json:"conversation_id"`
	AgentID        string        `json:"agent_id"`
	TenantID       string        `json:"tenant_id"`
	EndedAt        time.Time     `json:"ended_at"`
	Duration       time.Duration `json:"duration"`
	Reason         string        `json:"reason,omitempty"`
	EndedBy        string        `json:"ended_by,omitempty"`
}

// MessageSentEvent captures outbound content delivered by the agent.
type MessageSentEvent struct {
	MessageID      string    `json:"message_id"`
	ConversationID string    `json:"conversation_id"`
	AgentID        string    `json:"agent_id"`
	TenantID       string    `json:"tenant_id"`
	Content        string    `json:"content"`
	ContentType    string    `json:"content_type"`
	Channel        string    `json:"channel"`
	SentAt         time.Time `json:"sent_at"`
	Recipient      string    `json:"recipient,omitempty"`
}

// MessageReceivedEvent stores inbound messages that the agent processes.
type MessageReceivedEvent struct {
	MessageID      string    `json:"message_id"`
	ConversationID string    `json:"conversation_id"`
	AgentID        string    `json:"agent_id"`
	TenantID       string    `json:"tenant_id"`
	Content        string    `json:"content"`
	ContentType    string    `json:"content_type"`
	Channel        string    `json:"channel"`
	Source         string    `json:"source"`
	ReceivedAt     time.Time `json:"received_at"`
	Sender         string    `json:"sender,omitempty"`
}
