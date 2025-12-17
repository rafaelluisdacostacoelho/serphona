package events

import "time"

// SubscriptionCreatedEvent notifies about a brand-new subscription lifecycle.
type SubscriptionCreatedEvent struct {
	SubscriptionID string     `json:"subscription_id"`
	TenantID       string     `json:"tenant_id"`
	Plan           string     `json:"plan"`
	Status         string     `json:"status"`
	Seats          int        `json:"seats,omitempty"`
	Currency       string     `json:"currency,omitempty"`
	AmountCents    int64      `json:"amount_cents,omitempty"`
	StartedAt      time.Time  `json:"started_at"`
	TrialEndsAt    *time.Time `json:"trial_ends_at,omitempty"`
	CreatedBy      string     `json:"created_by,omitempty"`
}

// SubscriptionUpdatedEvent tracks modifications applied to a subscription record.
type SubscriptionUpdatedEvent struct {
	SubscriptionID string            `json:"subscription_id"`
	TenantID       string            `json:"tenant_id"`
	Changes        map[string]string `json:"changes,omitempty"`
	UpdatedAt      time.Time         `json:"updated_at"`
	UpdatedBy      string            `json:"updated_by,omitempty"`
}

// SubscriptionCancelledEvent signals a subscription cancellation workflow.
type SubscriptionCancelledEvent struct {
	SubscriptionID string    `json:"subscription_id"`
	TenantID       string    `json:"tenant_id"`
	Reason         string    `json:"reason,omitempty"`
	CancelledAt    time.Time `json:"cancelled_at"`
	CancelledBy    string    `json:"cancelled_by,omitempty"`
	Refunded       bool      `json:"refunded,omitempty"`
}

// PaymentSucceededEvent documents a successful payment collection.
type PaymentSucceededEvent struct {
	PaymentID   string    `json:"payment_id"`
	TenantID    string    `json:"tenant_id"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	InvoiceID   string    `json:"invoice_id,omitempty"`
	Method      string    `json:"method,omitempty"`
	Reference   string    `json:"reference,omitempty"`
	CapturedAt  time.Time `json:"captured_at"`
	CapturedBy  string    `json:"captured_by,omitempty"`
}

// PaymentFailedEvent keeps failure diagnostics for a payment attempt.
type PaymentFailedEvent struct {
	PaymentID     string    `json:"payment_id"`
	TenantID      string    `json:"tenant_id"`
	AmountCents   int64     `json:"amount_cents"`
	Currency      string    `json:"currency"`
	FailureCode   string    `json:"failure_code,omitempty"`
	FailureReason string    `json:"failure_reason,omitempty"`
	FailedAt      time.Time `json:"failed_at"`
	Retryable     bool      `json:"retryable,omitempty"`
}

// CreditsPurchasedEvent captures credit bundle purchases for prepaid consumption.
type CreditsPurchasedEvent struct {
	PurchaseID  string    `json:"purchase_id"`
	TenantID    string    `json:"tenant_id"`
	PackageID   string    `json:"package_id,omitempty"`
	Credits     int       `json:"credits"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	PurchasedAt time.Time `json:"purchased_at"`
	PurchasedBy string    `json:"purchased_by,omitempty"`
}

// CreditsConsumedEvent reflects credit deductions tied to product usage.
type CreditsConsumedEvent struct {
	UsageID    string    `json:"usage_id"`
	TenantID   string    `json:"tenant_id"`
	Credits    int       `json:"credits"`
	Reason     string    `json:"reason,omitempty"`
	Source     string    `json:"source,omitempty"`
	ConsumedAt time.Time `json:"consumed_at"`
}

// InvoiceGeneratedEvent informs downstream systems about a fresh invoice.
type InvoiceGeneratedEvent struct {
	InvoiceID   string    `json:"invoice_id"`
	TenantID    string    `json:"tenant_id"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	DueDate     time.Time `json:"due_date"`
	Link        string    `json:"link,omitempty"`
	GeneratedAt time.Time `json:"generated_at"`
	GeneratedBy string    `json:"generated_by,omitempty"`
}
