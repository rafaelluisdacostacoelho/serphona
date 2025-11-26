// Package tenant contains the tenant domain model and business logic.
package tenant

import (
	"time"

	"github.com/google/uuid"
)

// Status represents the tenant's current status.
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusPending   Status = "pending"
	StatusDeleted   Status = "deleted"
)

// Plan represents the tenant's subscription plan.
type Plan string

const (
	PlanStarter      Plan = "starter"
	PlanProfessional Plan = "professional"
	PlanEnterprise   Plan = "enterprise"
)

// Tenant represents a company/organization using the platform.
type Tenant struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Slug         string     `json:"slug"` // URL-friendly identifier
	Email        string     `json:"email"`
	Phone        string     `json:"phone,omitempty"`
	Status       Status     `json:"status"`
	Plan         Plan       `json:"plan"`
	Settings     Settings   `json:"settings"`
	Metadata     Metadata   `json:"metadata"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	StripeID     string     `json:"stripe_id,omitempty"`
	BillingEmail string     `json:"billing_email,omitempty"`
}

// Settings contains tenant-specific configuration.
type Settings struct {
	// Telephony settings
	Telephony TelephonySettings `json:"telephony"`
	// AI Agent settings
	AIAgent AIAgentSettings `json:"ai_agent"`
	// Notification settings
	Notifications NotificationSettings `json:"notifications"`
	// Security settings
	Security SecuritySettings `json:"security"`
}

// TelephonySettings contains telephony configuration.
type TelephonySettings struct {
	DefaultCountryCode   string   `json:"default_country_code"`
	AllowedCountries     []string `json:"allowed_countries"`
	RecordingEnabled     bool     `json:"recording_enabled"`
	TranscriptionEnabled bool     `json:"transcription_enabled"`
	MaxConcurrentCalls   int      `json:"max_concurrent_calls"`
	CallerIDNumber       string   `json:"caller_id_number,omitempty"`
	SIPTrunkID           string   `json:"sip_trunk_id,omitempty"`
}

// AIAgentSettings contains AI agent configuration.
type AIAgentSettings struct {
	DefaultLanguage     string            `json:"default_language"`
	DefaultVoice        string            `json:"default_voice"`
	SpeechModel         string            `json:"speech_model"`
	MaxConversationMin  int               `json:"max_conversation_min"`
	CustomPrompts       map[string]string `json:"custom_prompts,omitempty"`
	EnableSentiment     bool              `json:"enable_sentiment"`
	EnableSummarization bool              `json:"enable_summarization"`
}

// NotificationSettings contains notification preferences.
type NotificationSettings struct {
	EmailEnabled    bool     `json:"email_enabled"`
	WebhookEnabled  bool     `json:"webhook_enabled"`
	WebhookURL      string   `json:"webhook_url,omitempty"`
	WebhookSecret   string   `json:"-"` // Never expose in JSON
	SlackEnabled    bool     `json:"slack_enabled"`
	SlackWebhookURL string   `json:"slack_webhook_url,omitempty"`
	AlertRecipients []string `json:"alert_recipients,omitempty"`
}

// SecuritySettings contains security configuration.
type SecuritySettings struct {
	MFARequired       bool           `json:"mfa_required"`
	IPWhitelist       []string       `json:"ip_whitelist,omitempty"`
	AllowedDomains    []string       `json:"allowed_domains,omitempty"`
	SessionTimeoutMin int            `json:"session_timeout_min"`
	PasswordPolicy    PasswordPolicy `json:"password_policy"`
}

// PasswordPolicy defines password requirements.
type PasswordPolicy struct {
	MinLength        int  `json:"min_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireLowercase bool `json:"require_lowercase"`
	RequireNumbers   bool `json:"require_numbers"`
	RequireSpecial   bool `json:"require_special"`
}

// Metadata contains tenant metadata.
type Metadata struct {
	Industry    string            `json:"industry,omitempty"`
	CompanySize string            `json:"company_size,omitempty"`
	Website     string            `json:"website,omitempty"`
	Address     Address           `json:"address,omitempty"`
	Custom      map[string]string `json:"custom,omitempty"`
}

// Address represents a physical address.
type Address struct {
	Street     string `json:"street,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	Country    string `json:"country,omitempty"`
}

// Quota represents usage limits for a tenant.
type Quota struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	MaxAPIKeys         int       `json:"max_api_keys"`
	MaxUsers           int       `json:"max_users"`
	MaxCallsPerMonth   int       `json:"max_calls_per_month"`
	MaxMinutesPerMonth int       `json:"max_minutes_per_month"`
	MaxStorageGB       int       `json:"max_storage_gb"`
	UsedCalls          int       `json:"used_calls"`
	UsedMinutes        int       `json:"used_minutes"`
	UsedStorageGB      float64   `json:"used_storage_gb"`
	ResetAt            time.Time `json:"reset_at"`
}

// Usage represents current usage statistics.
type Usage struct {
	TenantID      uuid.UUID `json:"tenant_id"`
	Period        string    `json:"period"` // e.g., "2024-01"
	TotalCalls    int       `json:"total_calls"`
	TotalMinutes  int       `json:"total_minutes"`
	TotalMessages int       `json:"total_messages"`
	StorageUsedGB float64   `json:"storage_used_gb"`
	APIRequests   int64     `json:"api_requests"`
}

// NewTenant creates a new Tenant with default values.
func NewTenant(name, email string, plan Plan) *Tenant {
	now := time.Now().UTC()
	return &Tenant{
		ID:        uuid.New(),
		Name:      name,
		Email:     email,
		Status:    StatusPending,
		Plan:      plan,
		Settings:  DefaultSettings(),
		Metadata:  Metadata{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// DefaultSettings returns default tenant settings.
func DefaultSettings() Settings {
	return Settings{
		Telephony: TelephonySettings{
			DefaultCountryCode:   "US",
			AllowedCountries:     []string{"US", "CA", "GB"},
			RecordingEnabled:     true,
			TranscriptionEnabled: true,
			MaxConcurrentCalls:   10,
		},
		AIAgent: AIAgentSettings{
			DefaultLanguage:     "en-US",
			DefaultVoice:        "neural-standard",
			SpeechModel:         "whisper-large",
			MaxConversationMin:  30,
			EnableSentiment:     true,
			EnableSummarization: true,
		},
		Notifications: NotificationSettings{
			EmailEnabled: true,
		},
		Security: SecuritySettings{
			SessionTimeoutMin: 60,
			PasswordPolicy: PasswordPolicy{
				MinLength:        8,
				RequireUppercase: true,
				RequireLowercase: true,
				RequireNumbers:   true,
				RequireSpecial:   false,
			},
		},
	}
}

// Activate activates the tenant.
func (t *Tenant) Activate() {
	t.Status = StatusActive
	t.UpdatedAt = time.Now().UTC()
}

// Suspend suspends the tenant.
func (t *Tenant) Suspend() {
	t.Status = StatusSuspended
	t.UpdatedAt = time.Now().UTC()
}

// SoftDelete marks the tenant as deleted.
func (t *Tenant) SoftDelete() {
	now := time.Now().UTC()
	t.Status = StatusDeleted
	t.DeletedAt = &now
	t.UpdatedAt = now
}

// IsActive returns true if the tenant is active.
func (t *Tenant) IsActive() bool {
	return t.Status == StatusActive
}

// CanMakeCalls returns true if the tenant can make calls.
func (t *Tenant) CanMakeCalls() bool {
	return t.IsActive() && t.Settings.Telephony.MaxConcurrentCalls > 0
}
