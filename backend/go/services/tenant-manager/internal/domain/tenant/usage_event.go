package tenant

import (
	"time"

	"github.com/google/uuid"
)

// UsageReportedEvent represents the payload emitted to the usage.reported topic.
type UsageReportedEvent struct {
	TenantID       uuid.UUID `json:"tenant_id"`
	Period         string    `json:"period"`
	OccurredAt     time.Time `json:"occurred_at"`
	Source         string    `json:"source"`
	Calls          int       `json:"calls"`
	Minutes        int       `json:"minutes"`
	Messages       int       `json:"messages"`
	StorageGB      float64   `json:"storage_gb"`
	APIRequests    int       `json:"api_requests"`
	Plan           string    `json:"plan,omitempty"`
	SubscriptionID string    `json:"subscription_id,omitempty"`
	RequestID      string    `json:"request_id"`
	TraceID        string    `json:"trace_id,omitempty"`
}
