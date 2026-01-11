package audit

import "go.uber.org/zap"

// Event captures structured audit attributes.
type Event struct {
	Category string
	Action   string
	Outcome  string
	TenantID string
	UserID   string
	Service  string
	ToolID   string
	Path     string
	RuleID   string
	Reason   string
}

// Emit writes an audit event as structured log.
func Emit(log *zap.Logger, e Event) {
	fields := []zap.Field{
		zap.String("audit_category", e.Category),
		zap.String("audit_action", e.Action),
		zap.String("audit_outcome", e.Outcome),
		zap.String("tenant_id", e.TenantID),
	}
	if e.UserID != "" {
		fields = append(fields, zap.String("user_id", e.UserID))
	}
	if e.Service != "" {
		fields = append(fields, zap.String("service", e.Service))
	}
	if e.Path != "" {
		fields = append(fields, zap.String("path", e.Path))
	}
	if e.ToolID != "" {
		fields = append(fields, zap.String("tool_id", e.ToolID))
	}
	if e.RuleID != "" {
		fields = append(fields, zap.String("rule_id", e.RuleID))
	}
	if e.Reason != "" {
		fields = append(fields, zap.String("reason", e.Reason))
	}

	log.Info("audit", fields...)
}
