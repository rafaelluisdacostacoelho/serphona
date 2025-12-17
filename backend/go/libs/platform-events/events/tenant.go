package events

import "time"

// TenantCreatedEvent notifies downstream systems about a new tenant account.
type TenantCreatedEvent struct {
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug,omitempty"`
	Plan      string    `json:"plan"`
	OwnerID   string    `json:"owner_id"`
	Region    string    `json:"region,omitempty"`
	Status    string    `json:"status,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by,omitempty"`
}

// TenantUpdatedEvent highlights mutable tenant attributes that changed.
type TenantUpdatedEvent struct {
	TenantID  string            `json:"tenant_id"`
	Changes   map[string]string `json:"changes,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
	UpdatedBy string            `json:"updated_by,omitempty"`
}

// TenantDeletedEvent represents a soft or hard deletion of a tenant.
type TenantDeletedEvent struct {
	TenantID   string    `json:"tenant_id"`
	DeletedAt  time.Time `json:"deleted_at"`
	DeletedBy  string    `json:"deleted_by,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	HardDelete bool      `json:"hard_delete,omitempty"`
}

// TenantSuspendedEvent captures suspension details for tenant enforcement.
type TenantSuspendedEvent struct {
	TenantID    string     `json:"tenant_id"`
	SuspendedAt time.Time  `json:"suspended_at"`
	SuspendedBy string     `json:"suspended_by,omitempty"`
	Reason      string     `json:"reason"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// TenantActivatedEvent signals that a tenant is fully active again.
type TenantActivatedEvent struct {
	TenantID    string    `json:"tenant_id"`
	ActivatedAt time.Time `json:"activated_at"`
	ActivatedBy string    `json:"activated_by,omitempty"`
	Reason      string    `json:"reason,omitempty"`
}

// TenantMemberAddedEvent adds context about onboarding a tenant member.
type TenantMemberAddedEvent struct {
	TenantID     string    `json:"tenant_id"`
	MemberID     string    `json:"member_id"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	AddedAt      time.Time `json:"added_at"`
	AddedBy      string    `json:"added_by,omitempty"`
	InvitationID string    `json:"invitation_id,omitempty"`
	Source       string    `json:"source,omitempty"`
}

// TenantMemberRemovedEvent informs other services about member offboarding.
type TenantMemberRemovedEvent struct {
	TenantID  string    `json:"tenant_id"`
	MemberID  string    `json:"member_id"`
	Email     string    `json:"email,omitempty"`
	Role      string    `json:"role,omitempty"`
	RemovedAt time.Time `json:"removed_at"`
	RemovedBy string    `json:"removed_by,omitempty"`
	Reason    string    `json:"reason,omitempty"`
}
