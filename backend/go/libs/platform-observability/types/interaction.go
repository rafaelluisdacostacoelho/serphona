package types

import "time"

// Interaction representa uma interação em uma conversação
type Interaction struct {
	InteractionID  string            `json:"interaction_id"`
	ConversationID string            `json:"conversation_id"`
	TenantID       string            `json:"tenant_id"`
	Type           string            `json:"type"`    // agent_message, customer_message, system_message
	Speaker        string            `json:"speaker"` // agent, customer, system
	Content        string            `json:"content"`
	Timestamp      time.Time         `json:"timestamp"`
	Sentiment      string            `json:"sentiment,omitempty"` // positive, neutral, negative
	Intent         string            `json:"intent,omitempty"`
	Confidence     float64           `json:"confidence,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// InteractionEvent representa um evento de interação para logging
type InteractionEvent struct {
	InteractionID  string         `json:"interaction_id"`
	ConversationID string         `json:"conversation_id"`
	TenantID       string         `json:"tenant_id"`
	AgentID        string         `json:"agent_id"`
	CustomerID     string         `json:"customer_id"`
	EventType      string         `json:"event_type"` // interaction.agent, interaction.customer, interaction.system
	Speaker        string         `json:"speaker"`
	Content        string         `json:"content"`
	Timestamp      time.Time      `json:"timestamp"`
	Channel        string         `json:"channel"`
	Language       string         `json:"language"`
	Sentiment      string         `json:"sentiment,omitempty"`
	Intent         string         `json:"intent,omitempty"`
	Confidence     float64        `json:"confidence,omitempty"`
	Duration       time.Duration  `json:"duration,omitempty"` // tempo desde a última interação
	Metadata       map[string]any `json:"metadata,omitempty"`
	TraceID        string         `json:"trace_id,omitempty"`
	SpanID         string         `json:"span_id,omitempty"`
}

// InteractionAnalysis representa a análise de uma interação
type InteractionAnalysis struct {
	InteractionID string            `json:"interaction_id"`
	Sentiment     SentimentAnalysis `json:"sentiment"`
	Intent        IntentAnalysis    `json:"intent"`
	Entities      []Entity          `json:"entities"`
	Topics        []string          `json:"topics"`
	Language      string            `json:"language"`
	Toxicity      float64           `json:"toxicity"`
	Urgency       float64           `json:"urgency"`
	Complexity    float64           `json:"complexity"`
}

// SentimentAnalysis representa análise de sentimento
type SentimentAnalysis struct {
	Label      string  `json:"label"` // positive, neutral, negative
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"`
}

// IntentAnalysis representa análise de intenção
type IntentAnalysis struct {
	Label      string  `json:"label"` // greeting, complaint, question, request, etc
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"`
}

// Entity representa uma entidade detectada na interação
type Entity struct {
	Type       string  `json:"type"` // person, location, organization, product, etc
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
}

// InteractionMetrics representa métricas de interações
type InteractionMetrics struct {
	TenantID               string           `json:"tenant_id"`
	Period                 time.Time        `json:"period"`
	TotalInteractions      int64            `json:"total_interactions"`
	AveragePerConversation float64          `json:"average_per_conversation"`
	AverageResponseTime    time.Duration    `json:"average_response_time"`
	SentimentDistribution  map[string]int64 `json:"sentiment_distribution"`
	IntentDistribution     map[string]int64 `json:"intent_distribution"`
	ChannelDistribution    map[string]int64 `json:"channel_distribution"`
	TopEntities            []EntityCount    `json:"top_entities"`
}

// EntityCount representa contagem de uma entidade
type EntityCount struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Count int64  `json:"count"`
}
