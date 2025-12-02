package types

import "time"

// Decision representa uma decisão tomada durante a conversação
type Decision struct {
	DecisionID     string            `json:"decision_id"`
	ConversationID string            `json:"conversation_id"`
	TenantID       string            `json:"tenant_id"`
	AgentID        string            `json:"agent_id"`
	DecisionType   string            `json:"decision_type"` // transfer, escalate, close, offer, reject, etc
	Option         string            `json:"option"`
	Reason         string            `json:"reason,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
	Context        map[string]string `json:"context,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// DecisionEvent representa um evento de decisão para logging
type DecisionEvent struct {
	DecisionID     string         `json:"decision_id"`
	ConversationID string         `json:"conversation_id"`
	TenantID       string         `json:"tenant_id"`
	AgentID        string         `json:"agent_id"`
	CustomerID     string         `json:"customer_id"`
	EventType      string         `json:"event_type"` // decision.made, decision.reverted
	DecisionType   string         `json:"decision_type"`
	Option         string         `json:"option"`
	Reason         string         `json:"reason,omitempty"`
	Timestamp      time.Time      `json:"timestamp"`
	Context        map[string]any `json:"context,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	TraceID        string         `json:"trace_id,omitempty"`
	SpanID         string         `json:"span_id,omitempty"`
}

// DecisionMetrics representa métricas de decisões
type DecisionMetrics struct {
	TenantID              string                `json:"tenant_id"`
	Period                time.Time             `json:"period"`
	TotalDecisions        int64                 `json:"total_decisions"`
	DecisionDistribution  map[string]int64      `json:"decision_distribution"`
	TopAgents             []AgentDecisionMetric `json:"top_agents"`
	AverageTimeToDecision time.Duration         `json:"average_time_to_decision"`
}

// AgentDecisionMetric representa métricas de decisões de um agente
type AgentDecisionMetric struct {
	AgentID               string           `json:"agent_id"`
	TotalDecisions        int64            `json:"total_decisions"`
	DecisionTypes         map[string]int64 `json:"decision_types"`
	AverageTimeToDecision time.Duration    `json:"average_time_to_decision"`
	SuccessRate           float64          `json:"success_rate"`
}

// TransferDecision representa uma decisão de transferência
type TransferDecision struct {
	Decision
	FromAgentID     string     `json:"from_agent_id"`
	ToAgentID       string     `json:"to_agent_id,omitempty"`
	ToDepartment    string     `json:"to_department,omitempty"`
	TransferType    string     `json:"transfer_type"` // warm, cold, blind
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
	RejectedAt      *time.Time `json:"rejected_at,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
}

// EscalationDecision representa uma decisão de escalação
type EscalationDecision struct {
	Decision
	EscalationLevel int        `json:"escalation_level"` // 1, 2, 3
	Priority        string     `json:"priority"`         // low, medium, high, critical
	AssignedTo      string     `json:"assigned_to,omitempty"`
	DueDate         *time.Time `json:"due_date,omitempty"`
}

// OfferDecision representa uma decisão de oferta
type OfferDecision struct {
	Decision
	OfferType  string     `json:"offer_type"` // discount, upgrade, refund, etc
	OfferValue float64    `json:"offer_value,omitempty"`
	Currency   string     `json:"currency,omitempty"`
	Accepted   bool       `json:"accepted"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	RejectedAt *time.Time `json:"rejected_at,omitempty"`
}

// ComplianceCheck representa uma verificação de compliance
type ComplianceCheck struct {
	CheckID        string            `json:"check_id"`
	ConversationID string            `json:"conversation_id"`
	TenantID       string            `json:"tenant_id"`
	PolicyID       string            `json:"policy_id"`
	PolicyName     string            `json:"policy_name"`
	Timestamp      time.Time         `json:"timestamp"`
	Passed         bool              `json:"passed"`
	Score          float64           `json:"score"`
	Violations     []Violation       `json:"violations,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// Violation representa uma violação de compliance
type Violation struct {
	ViolationID string    `json:"violation_id"`
	RuleID      string    `json:"rule_id"`
	RuleName    string    `json:"rule_name"`
	Severity    string    `json:"severity"` // low, medium, high, critical
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Context     string    `json:"context,omitempty"`
}
