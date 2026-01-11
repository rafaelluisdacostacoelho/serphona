package audit

import (
	"math/rand"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Event captures structured audit attributes.
type Event struct {
	Category  string
	Action    string
	Outcome   string
	TenantID  string
	UserID    string
	Service   string
	ToolID    string
	VersionID string
	Path      string
	RuleID    string
	Reason    string
	DiffHash  string
}

var sampleRate atomic.Value

func init() {
	rand.Seed(time.Now().UnixNano())
	sampleRate.Store(1.0)
}

// SetSampleRate configures audit sampling between 0 and 1.
func SetSampleRate(rate float64) {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	sampleRate.Store(rate)
}

// Emit writes an audit event as structured log.
func Emit(log *zap.Logger, e Event) {
	if r, ok := sampleRate.Load().(float64); ok && r < 1 {
		if rand.Float64() > r {
			return
		}
	}

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
	if e.VersionID != "" {
		fields = append(fields, zap.String("tool_version_id", e.VersionID))
	}
	if e.RuleID != "" {
		fields = append(fields, zap.String("rule_id", e.RuleID))
	}
	if e.Reason != "" {
		fields = append(fields, zap.String("reason", e.Reason))
	}
	if e.DiffHash != "" {
		fields = append(fields, zap.String("diff_hash", e.DiffHash))
	}

	log.Info("audit", fields...)
}
