package events

import "time"

// UserCreatedEvent captures metadata for a brand-new user.
type UserCreatedEvent struct {
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by,omitempty"`
}

// UserUpdatedEvent records profile or role changes for an existing user.
type UserUpdatedEvent struct {
	UserID    string            `json:"user_id"`
	TenantID  string            `json:"tenant_id"`
	Changes   map[string]string `json:"changes,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
	UpdatedBy string            `json:"updated_by,omitempty"`
}

// UserDeletedEvent signals that a user account was removed.
type UserDeletedEvent struct {
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	DeletedAt time.Time `json:"deleted_at"`
	DeletedBy string    `json:"deleted_by,omitempty"`
	Reason    string    `json:"reason,omitempty"`
}

// UserLoggedInEvent keeps audit information for successful sign-ins.
type UserLoggedInEvent struct {
	UserID     string    `json:"user_id"`
	TenantID   string    `json:"tenant_id"`
	LoggedInAt time.Time `json:"logged_in_at"`
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	Method     string    `json:"method,omitempty"`
}

// UserLoggedOutEvent registers user sessions that ended voluntarily or otherwise.
type UserLoggedOutEvent struct {
	UserID      string    `json:"user_id"`
	TenantID    string    `json:"tenant_id"`
	LoggedOutAt time.Time `json:"logged_out_at"`
	Method      string    `json:"method,omitempty"`
	Reason      string    `json:"reason,omitempty"`
}

// PasswordChangedEvent captures credential rotations initiated by the user or admin.
type PasswordChangedEvent struct {
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	ChangedAt time.Time `json:"changed_at"`
	ChangedBy string    `json:"changed_by,omitempty"`
	Method    string    `json:"method,omitempty"`
}

// PasswordResetEvent documents full password reset flows.
type PasswordResetEvent struct {
	UserID   string    `json:"user_id"`
	TenantID string    `json:"tenant_id"`
	ResetAt  time.Time `json:"reset_at"`
	ResetBy  string    `json:"reset_by,omitempty"`
	Method   string    `json:"method,omitempty"`
	TokenID  string    `json:"token_id,omitempty"`
}
