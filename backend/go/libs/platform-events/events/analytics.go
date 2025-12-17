package events

import "time"

// InteractionLoggedEvent captures raw interaction facts for analytics pipelines.
type InteractionLoggedEvent struct {
	InteractionID string                 `json:"interaction_id"`
	TenantID      string                 `json:"tenant_id"`
	UserID        string                 `json:"user_id,omitempty"`
	AgentID       string                 `json:"agent_id,omitempty"`
	Channel       string                 `json:"channel"`
	LoggedAt      time.Time              `json:"logged_at"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// MetricRecordedEvent stores point-in-time metric observations with dimensions.
type MetricRecordedEvent struct {
	Metric     string            `json:"metric"`
	TenantID   string            `json:"tenant_id"`
	Value      float64           `json:"value"`
	Unit       string            `json:"unit,omitempty"`
	Dimensions map[string]string `json:"dimensions,omitempty"`
	CapturedAt time.Time         `json:"captured_at"`
	Source     string            `json:"source,omitempty"`
}

// ReportGeneratedEvent signals the availability of a generated analytics report.
type ReportGeneratedEvent struct {
	ReportID    string    `json:"report_id"`
	TenantID    string    `json:"tenant_id"`
	ReportType  string    `json:"report_type"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	GeneratedAt time.Time `json:"generated_at"`
	Link        string    `json:"link,omitempty"`
	RequestedBy string    `json:"requested_by,omitempty"`
}

// DataExportedEvent records exports produced for analytics deliveries.
type DataExportedEvent struct {
	ExportID    string    `json:"export_id"`
	TenantID    string    `json:"tenant_id"`
	Format      string    `json:"format"`
	Destination string    `json:"destination"`
	SizeBytes   int64     `json:"size_bytes,omitempty"`
	RequestedBy string    `json:"requested_by,omitempty"`
	ExportedAt  time.Time `json:"exported_at"`
	Status      string    `json:"status,omitempty"`
}
