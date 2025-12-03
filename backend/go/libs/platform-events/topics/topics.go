package topics

// Tópicos padrão do sistema Serphona
const (
	// Auth events
	UserCreated     = "auth.user.created"
	UserUpdated     = "auth.user.updated"
	UserDeleted     = "auth.user.deleted"
	UserLoggedIn    = "auth.user.logged_in"
	UserLoggedOut   = "auth.user.logged_out"
	PasswordChanged = "auth.password.changed"
	PasswordReset   = "auth.password.reset"

	// Tenant events
	TenantCreated       = "tenant.created"
	TenantUpdated       = "tenant.updated"
	TenantDeleted       = "tenant.deleted"
	TenantSuspended     = "tenant.suspended"
	TenantActivated     = "tenant.activated"
	TenantMemberAdded   = "tenant.member.added"
	TenantMemberRemoved = "tenant.member.removed"

	// Billing events
	SubscriptionCreated   = "billing.subscription.created"
	SubscriptionUpdated   = "billing.subscription.updated"
	SubscriptionCancelled = "billing.subscription.cancelled"
	PaymentSucceeded      = "billing.payment.succeeded"
	PaymentFailed         = "billing.payment.failed"
	CreditsPurchased      = "billing.credits.purchased"
	CreditsConsumed       = "billing.credits.consumed"
	InvoiceGenerated      = "billing.invoice.generated"

	// Agent events
	AgentCreated        = "agent.created"
	AgentUpdated        = "agent.updated"
	AgentDeleted        = "agent.deleted"
	AgentDeployed       = "agent.deployed"
	AgentStarted        = "agent.started"
	AgentStopped        = "agent.stopped"
	ConversationStarted = "agent.conversation.started"
	ConversationEnded   = "agent.conversation.ended"
	MessageSent         = "agent.message.sent"
	MessageReceived     = "agent.message.received"

	// Analytics events
	InteractionLogged = "analytics.interaction.logged"
	MetricRecorded    = "analytics.metric.recorded"
	ReportGenerated   = "analytics.report.generated"
	DataExported      = "analytics.data.exported"

	// Tool events
	ToolRegistered = "tool.registered"
	ToolInvoked    = "tool.invoked"
	ToolCompleted  = "tool.completed"
	ToolFailed     = "tool.failed"

	// System events
	SystemHealthCheck    = "system.health.check"
	SystemError          = "system.error"
	SystemAlert          = "system.alert"
	ConfigurationUpdated = "system.configuration.updated"
)

// TopicGroups agrupa tópicos por categoria
var TopicGroups = map[string][]string{
	"auth": {
		UserCreated,
		UserUpdated,
		UserDeleted,
		UserLoggedIn,
		UserLoggedOut,
		PasswordChanged,
		PasswordReset,
	},
	"tenant": {
		TenantCreated,
		TenantUpdated,
		TenantDeleted,
		TenantSuspended,
		TenantActivated,
		TenantMemberAdded,
		TenantMemberRemoved,
	},
	"billing": {
		SubscriptionCreated,
		SubscriptionUpdated,
		SubscriptionCancelled,
		PaymentSucceeded,
		PaymentFailed,
		CreditsPurchased,
		CreditsConsumed,
		InvoiceGenerated,
	},
	"agent": {
		AgentCreated,
		AgentUpdated,
		AgentDeleted,
		AgentDeployed,
		AgentStarted,
		AgentStopped,
		ConversationStarted,
		ConversationEnded,
		MessageSent,
		MessageReceived,
	},
	"analytics": {
		InteractionLogged,
		MetricRecorded,
		ReportGenerated,
		DataExported,
	},
	"tool": {
		ToolRegistered,
		ToolInvoked,
		ToolCompleted,
		ToolFailed,
	},
	"system": {
		SystemHealthCheck,
		SystemError,
		SystemAlert,
		ConfigurationUpdated,
	},
}

// GetTopicsByGroup retorna todos os tópicos de um grupo
func GetTopicsByGroup(group string) []string {
	return TopicGroups[group]
}

// AllTopics retorna todos os tópicos disponíveis
func AllTopics() []string {
	var topics []string
	for _, group := range TopicGroups {
		topics = append(topics, group...)
	}
	return topics
}
