package types

import "time"

// ConversationStart representa o início de uma conversação
type ConversationStart struct {
	ConversationID string            `json:"conversation_id"`
	TenantID       string            `json:"tenant_id"`
	AgentID        string            `json:"agent_id"`
	CustomerID     string            `json:"customer_id"`
	Channel        string            `json:"channel"` // voice, chat, email, whatsapp
	Language       string            `json:"language"`
	StartTime      time.Time         `json:"start_time"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// ConversationEnd representa o fim de uma conversação
type ConversationEnd struct {
	ConversationID string            `json:"conversation_id"`
	EndTime        time.Time         `json:"end_time"`
	Duration       time.Duration     `json:"duration"`
	Resolution     string            `json:"resolution"` // solved, transferred, abandoned, escalated
	Rating         int               `json:"rating"`     // 1-5
	Tags           []string          `json:"tags"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// Conversation representa uma conversação completa
type Conversation struct {
	ConversationID   string         `json:"conversation_id"`
	TenantID         string         `json:"tenant_id"`
	AgentID          string         `json:"agent_id"`
	CustomerID       string         `json:"customer_id"`
	Channel          string         `json:"channel"`
	Language         string         `json:"language"`
	StartTime        time.Time      `json:"start_time"`
	EndTime          time.Time      `json:"end_time"`
	Duration         time.Duration  `json:"duration"`
	InteractionCount int            `json:"interaction_count"`
	Interactions     []Interaction  `json:"interactions"`
	Decisions        []Decision     `json:"decisions"`
	Resolution       string         `json:"resolution"`
	Rating           int            `json:"rating"`
	Tags             []string       `json:"tags"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

// ConversationMetrics representa métricas agregadas de conversações
type ConversationMetrics struct {
	TenantID               string           `json:"tenant_id"`
	Period                 time.Time        `json:"period"`
	TotalConversations     int64            `json:"total_conversations"`
	AverageDuration        time.Duration    `json:"average_duration"`
	AverageRating          float64          `json:"average_rating"`
	ResolutionRate         float64          `json:"resolution_rate"`
	TransferRate           float64          `json:"transfer_rate"`
	AbandonmentRate        float64          `json:"abandonment_rate"`
	AverageSentiment       float64          `json:"average_sentiment"`
	ComplianceScore        float64          `json:"compliance_score"`
	TopAgents              []AgentMetric    `json:"top_agents"`
	TopIntents             []string         `json:"top_intents"`
	ChannelDistribution    map[string]int64 `json:"channel_distribution"`
	ResolutionDistribution map[string]int64 `json:"resolution_distribution"`
}

// AgentMetric representa métricas de um agente
type AgentMetric struct {
	AgentID             string        `json:"agent_id"`
	ConversationCount   int64         `json:"conversation_count"`
	AverageDuration     time.Duration `json:"average_duration"`
	AverageRating       float64       `json:"average_rating"`
	ResolutionRate      float64       `json:"resolution_rate"`
	AverageResponseTime time.Duration `json:"average_response_time"`
}
