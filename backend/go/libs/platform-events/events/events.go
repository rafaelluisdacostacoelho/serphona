package events

import (
	"time"

	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

// UserCreatedEvent representa um evento de criação de usuário
type UserCreatedEvent struct {
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// UserUpdatedEvent representa um evento de atualização de usuário
type UserUpdatedEvent struct {
	UserID    string            `json:"user_id"`
	TenantID  string            `json:"tenant_id"`
	Changes   map[string]string `json:"changes"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// TenantCreatedEvent representa um evento de criação de tenant
type TenantCreatedEvent struct {
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantUpdatedEvent representa um evento de atualização de tenant
type TenantUpdatedEvent struct {
	TenantID  string            `json:"tenant_id"`
	Changes   map[string]string `json:"changes"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// SubscriptionCreatedEvent representa um evento de criação de assinatura
type SubscriptionCreatedEvent struct {
	SubscriptionID string    `json:"subscription_id"`
	TenantID       string    `json:"tenant_id"`
	Plan           string    `json:"plan"`
	Status         string    `json:"status"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
}

// PaymentSucceededEvent representa um evento de pagamento bem-sucedido
type PaymentSucceededEvent struct {
	PaymentID      string    `json:"payment_id"`
	TenantID       string    `json:"tenant_id"`
	SubscriptionID string    `json:"subscription_id,omitempty"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Method         string    `json:"method"`
	PaidAt         time.Time `json:"paid_at"`
}

// CreditsPurchasedEvent representa um evento de compra de créditos
type CreditsPurchasedEvent struct {
	TransactionID string    `json:"transaction_id"`
	TenantID      string    `json:"tenant_id"`
	UserID        string    `json:"user_id"`
	Credits       int       `json:"credits"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	PurchasedAt   time.Time `json:"purchased_at"`
}

// CreditsConsumedEvent representa um evento de consumo de créditos
type CreditsConsumedEvent struct {
	TenantID     string    `json:"tenant_id"`
	UserID       string    `json:"user_id"`
	Credits      int       `json:"credits"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	ConsumedAt   time.Time `json:"consumed_at"`
}

// AgentCreatedEvent representa um evento de criação de agente
type AgentCreatedEvent struct {
	AgentID   string    `json:"agent_id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Model     string    `json:"model"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// ConversationStartedEvent representa um evento de início de conversa
type ConversationStartedEvent struct {
	ConversationID string    `json:"conversation_id"`
	AgentID        string    `json:"agent_id"`
	TenantID       string    `json:"tenant_id"`
	CustomerID     string    `json:"customer_id"`
	Channel        string    `json:"channel"`
	Language       string    `json:"language"`
	StartedAt      time.Time `json:"started_at"`
}

// ConversationEndedEvent representa um evento de fim de conversa
type ConversationEndedEvent struct {
	ConversationID string        `json:"conversation_id"`
	AgentID        string        `json:"agent_id"`
	TenantID       string        `json:"tenant_id"`
	Duration       time.Duration `json:"duration"`
	MessageCount   int           `json:"message_count"`
	Resolution     string        `json:"resolution"`
	CustomerRating int           `json:"customer_rating,omitempty"`
	EndedAt        time.Time     `json:"ended_at"`
}

// MessageSentEvent representa um evento de mensagem enviada
type MessageSentEvent struct {
	MessageID      string                 `json:"message_id"`
	ConversationID string                 `json:"conversation_id"`
	AgentID        string                 `json:"agent_id"`
	TenantID       string                 `json:"tenant_id"`
	Content        string                 `json:"content"`
	Type           string                 `json:"type"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	SentAt         time.Time              `json:"sent_at"`
}

// ToolInvokedEvent representa um evento de invocação de ferramenta
type ToolInvokedEvent struct {
	ToolID         string                 `json:"tool_id"`
	TenantID       string                 `json:"tenant_id"`
	AgentID        string                 `json:"agent_id,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	Action         string                 `json:"action"`
	Parameters     map[string]interface{} `json:"parameters"`
	InvokedAt      time.Time              `json:"invoked_at"`
}

// ToolCompletedEvent representa um evento de conclusão de ferramenta
type ToolCompletedEvent struct {
	ToolID         string                 `json:"tool_id"`
	TenantID       string                 `json:"tenant_id"`
	AgentID        string                 `json:"agent_id,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	Action         string                 `json:"action"`
	Result         map[string]interface{} `json:"result"`
	Duration       time.Duration          `json:"duration"`
	CompletedAt    time.Time              `json:"completed_at"`
}

// SystemErrorEvent representa um evento de erro do sistema
type SystemErrorEvent struct {
	ErrorID    string                 `json:"error_id"`
	Service    string                 `json:"service"`
	Severity   string                 `json:"severity"`
	Message    string                 `json:"message"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	OccurredAt time.Time              `json:"occurred_at"`
}

// NewEvent é um helper para criar eventos com tipo e dados
func NewEvent(eventType, source string, data interface{}) *types.Event {
	return types.NewEvent(eventType, source, data)
}
