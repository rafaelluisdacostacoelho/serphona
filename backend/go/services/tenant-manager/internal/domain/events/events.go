// Package events contains domain events for the tenant manager.
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of domain event.
type EventType string

const (
	// Tenant events
	EventTypeTenantCreated     EventType = "tenant.created"
	EventTypeTenantUpdated     EventType = "tenant.updated"
	EventTypeTenantActivated   EventType = "tenant.activated"
	EventTypeTenantSuspended   EventType = "tenant.suspended"
	EventTypeTenantDeleted     EventType = "tenant.deleted"
	EventTypeTenantPlanChanged EventType = "tenant.plan_changed"

	// API Key events
	EventTypeAPIKeyCreated EventType = "apikey.created"
	EventTypeAPIKeyRevoked EventType = "apikey.revoked"
	EventTypeAPIKeyExpired EventType = "apikey.expired"
	EventTypeAPIKeyUsed    EventType = "apikey.used"

	// Config events
	EventTypeConfigUpdated EventType = "config.updated"
)

// Event is the base interface for all domain events.
type Event interface {
	GetID() uuid.UUID
	GetType() EventType
	GetTenantID() uuid.UUID
	GetTimestamp() time.Time
	GetVersion() string
	ToJSON() ([]byte, error)
}

// BaseEvent contains common fields for all events.
type BaseEvent struct {
	ID        uuid.UUID `json:"id"`
	Type      EventType `json:"type"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

// GetID returns the event ID.
func (e *BaseEvent) GetID() uuid.UUID {
	return e.ID
}

// GetType returns the event type.
func (e *BaseEvent) GetType() EventType {
	return e.Type
}

// GetTenantID returns the tenant ID.
func (e *BaseEvent) GetTenantID() uuid.UUID {
	return e.TenantID
}

// GetTimestamp returns the event timestamp.
func (e *BaseEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// GetVersion returns the event version.
func (e *BaseEvent) GetVersion() string {
	return e.Version
}

// NewBaseEvent creates a new base event.
func NewBaseEvent(eventType EventType, tenantID uuid.UUID) BaseEvent {
	return BaseEvent{
		ID:        uuid.New(),
		Type:      eventType,
		TenantID:  tenantID,
		Timestamp: time.Now().UTC(),
		Version:   "1.0",
	}
}

// TenantCreatedEvent represents a tenant creation event.
type TenantCreatedEvent struct {
	BaseEvent
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Email     string `json:"email"`
	Plan      string `json:"plan"`
	Status    string `json:"status"`
	CreatedBy string `json:"created_by,omitempty"`
}

// ToJSON converts the event to JSON.
func (e *TenantCreatedEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// NewTenantCreatedEvent creates a new tenant created event.
func NewTenantCreatedEvent(tenantID uuid.UUID, name, slug, email, plan, status, createdBy string) *TenantCreatedEvent {
	return &TenantCreatedEvent{
		BaseEvent: NewBaseEvent(EventTypeTenantCreated, tenantID),
		Name:      name,
		Slug:      slug,
		Email:     email,
		Plan:      plan,
		Status:    status,
		CreatedBy: createdBy,
	}
}

// TenantUpdatedEvent represents a tenant update event.
type TenantUpdatedEvent struct {
	BaseEvent
	Name      string            `json:"name"`
	Email     string            `json:"email"`
	Phone     string            `json:"phone,omitempty"`
	UpdatedBy string            `json:"updated_by,omitempty"`
	Changes   map[string]string `json:"changes,omitempty"`
}

// ToJSON converts the event to JSON.
func (e *TenantUpdatedEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// NewTenantUpdatedEvent creates a new tenant updated event.
func NewTenantUpdatedEvent(tenantID uuid.UUID, name, email, phone, updatedBy string, changes map[string]string) *TenantUpdatedEvent {
	return &TenantUpdatedEvent{
		BaseEvent: NewBaseEvent(EventTypeTenantUpdated, tenantID),
		Name:      name,
		Email:     email,
		Phone:     phone,
		UpdatedBy: updatedBy,
		Changes:   changes,
	}
}

// TenantActivatedEvent represents a tenant activation event.
type TenantActivatedEvent struct {
	BaseEvent
	ActivatedBy string `json:"activated_by,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// ToJSON converts the event to JSON.
func (e *TenantActivatedEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// NewTenantActivatedEvent creates a new tenant activated event.
func NewTenantActivatedEvent(tenantID uuid.UUID, activatedBy, reason string) *TenantActivatedEvent {
	return &TenantActivatedEvent{
		BaseEvent:   NewBaseEvent(EventTypeTenantActivated, tenantID),
		ActivatedBy: activatedBy,
		Reason:      reason,
	}
}

// TenantSuspendedEvent represents a tenant suspension event.
type TenantSuspendedEvent struct {
	BaseEvent
	SuspendedBy string `json:"suspended_by,omitempty"`
	Reason      string `json:"reason"`
}

// ToJSON converts the event to JSON.
func (e *TenantSuspendedEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// NewTenantSuspendedEvent creates a new tenant suspended event.
func NewTenantSuspendedEvent(tenantID uuid.UUID, suspendedBy, reason string) *TenantSuspendedEvent {
	return &TenantSuspendedEvent{
		BaseEvent:   NewBaseEvent(EventTypeTenantSuspended, tenantID),
		SuspendedBy: suspendedBy,
		Reason:      reason,
	}
}

// TenantDeletedEvent represents a tenant deletion event.
type TenantDeletedEvent struct {
	BaseEvent
	DeletedBy string `json:"deleted_by,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// ToJSON converts the event to JSON.
func (e *TenantDeletedEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// NewTenantDeletedEvent creates a new tenant deleted event.
func NewTenantDeletedEvent(tenantID uuid.UUID, deletedBy, reason string) *TenantDeletedEvent {
	return &TenantDeletedEvent{
		BaseEvent: NewBaseEvent(EventTypeTenantDeleted, tenantID),
		DeletedBy: deletedBy,
		Reason:    reason,
	}
}

// TenantPlanChangedEvent represents a tenant plan change event.
type TenantPlanChangedEvent struct {
	BaseEvent
	OldPlan   string `json:"old_plan"`
	NewPlan   string `json:"new_plan"`
	ChangedBy string `json:"changed_by,omitempty"`
}

// ToJSON converts the event to JSON.
func (e *TenantPlanChangedEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// NewTenantPlanChangedEvent creates a new tenant plan changed event.
func NewTenantPlanChangedEvent(tenantID uuid.UUID, oldPlan, newPlan, changedBy string) *TenantPlanChangedEvent {
	return &TenantPlanChangedEvent{
		BaseEvent: NewBaseEvent(EventTypeTenantPlanChanged, tenantID),
		OldPlan:   oldPlan,
		NewPlan:   newPlan,
		ChangedBy: changedBy,
	}
}

// APIKeyCreatedEvent represents an API key creation event.
type APIKeyCreatedEvent struct {
	BaseEvent
	KeyID       uuid.UUID `json:"key_id"`
	KeyName     string    `json:"key_name"`
	KeyPrefix   string    `json:"key_prefix"`
	Permissions []string  `json:"permissions"`
	ExpiresAt   string    `json:"expires_at,omitempty"`
	CreatedBy   uuid.UUID `json:"created_by"`
}

// ToJSON converts the event to JSON.
func (e *APIKeyCreatedEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// NewAPIKeyCreatedEvent creates a new API key created event.
func NewAPIKeyCreatedEvent(tenantID, keyID, createdBy uuid.UUID, keyName, keyPrefix string, permissions []string, expiresAt string) *APIKeyCreatedEvent {
	return &APIKeyCreatedEvent{
		BaseEvent:   NewBaseEvent(EventTypeAPIKeyCreated, tenantID),
		KeyID:       keyID,
		KeyName:     keyName,
		KeyPrefix:   keyPrefix,
		Permissions: permissions,
		ExpiresAt:   expiresAt,
		CreatedBy:   createdBy,
	}
}

// APIKeyRevokedEvent represents an API key revocation event.
type APIKeyRevokedEvent struct {
	BaseEvent
	KeyID     uuid.UUID `json:"key_id"`
	KeyName   string    `json:"key_name"`
	KeyPrefix string    `json:"key_prefix"`
	RevokedBy uuid.UUID `json:"revoked_by"`
	Reason    string    `json:"reason,omitempty"`
}

// ToJSON converts the event to JSON.
func (e *APIKeyRevokedEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// NewAPIKeyRevokedEvent creates a new API key revoked event.
func NewAPIKeyRevokedEvent(tenantID, keyID, revokedBy uuid.UUID, keyName, keyPrefix, reason string) *APIKeyRevokedEvent {
	return &APIKeyRevokedEvent{
		BaseEvent: NewBaseEvent(EventTypeAPIKeyRevoked, tenantID),
		KeyID:     keyID,
		KeyName:   keyName,
		KeyPrefix: keyPrefix,
		RevokedBy: revokedBy,
		Reason:    reason,
	}
}

// APIKeyExpiredEvent represents an API key expiration event.
type APIKeyExpiredEvent struct {
	BaseEvent
	KeyID     uuid.UUID `json:"key_id"`
	KeyName   string    `json:"key_name"`
	KeyPrefix string    `json:"key_prefix"`
	ExpiredAt string    `json:"expired_at"`
}

// ToJSON converts the event to JSON.
func (e *APIKeyExpiredEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// NewAPIKeyExpiredEvent creates a new API key expired event.
func NewAPIKeyExpiredEvent(tenantID, keyID uuid.UUID, keyName, keyPrefix, expiredAt string) *APIKeyExpiredEvent {
	return &APIKeyExpiredEvent{
		BaseEvent: NewBaseEvent(EventTypeAPIKeyExpired, tenantID),
		KeyID:     keyID,
		KeyName:   keyName,
		KeyPrefix: keyPrefix,
		ExpiredAt: expiredAt,
	}
}

// APIKeyUsedEvent represents an API key usage event.
type APIKeyUsedEvent struct {
	BaseEvent
	KeyID       uuid.UUID `json:"key_id"`
	KeyPrefix   string    `json:"key_prefix"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent,omitempty"`
	Success     bool      `json:"success"`
	ErrorReason string    `json:"error_reason,omitempty"`
}

// ToJSON converts the event to JSON.
func (e *APIKeyUsedEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// NewAPIKeyUsedEvent creates a new API key used event.
func NewAPIKeyUsedEvent(tenantID, keyID uuid.UUID, keyPrefix, ipAddress, userAgent string, success bool, errorReason string) *APIKeyUsedEvent {
	return &APIKeyUsedEvent{
		BaseEvent:   NewBaseEvent(EventTypeAPIKeyUsed, tenantID),
		KeyID:       keyID,
		KeyPrefix:   keyPrefix,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Success:     success,
		ErrorReason: errorReason,
	}
}

// ConfigUpdatedEvent represents a configuration update event.
type ConfigUpdatedEvent struct {
	BaseEvent
	ConfigType string            `json:"config_type"` // e.g., "telephony", "ai_agent", "security"
	Changes    map[string]string `json:"changes"`
	UpdatedBy  string            `json:"updated_by,omitempty"`
}

// ToJSON converts the event to JSON.
func (e *ConfigUpdatedEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// NewConfigUpdatedEvent creates a new config updated event.
func NewConfigUpdatedEvent(tenantID uuid.UUID, configType string, changes map[string]string, updatedBy string) *ConfigUpdatedEvent {
	return &ConfigUpdatedEvent{
		BaseEvent:  NewBaseEvent(EventTypeConfigUpdated, tenantID),
		ConfigType: configType,
		Changes:    changes,
		UpdatedBy:  updatedBy,
	}
}
