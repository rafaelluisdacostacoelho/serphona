package user

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email      string    `gorm:"uniqueIndex;not null"`
	Password   string    `gorm:"not null"` // Hashed password
	Name       string    `gorm:"not null"`
	Role       string    `gorm:"not null;default:'user'"` // admin, user, viewer
	TenantID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Provider   string    `gorm:"default:'local'"` // local, google, apple, microsoft
	ProviderID string    `gorm:"uniqueIndex:idx_provider_id,where:provider != 'local'"`
	Verified   bool      `gorm:"default:false"`
	Active     bool      `gorm:"default:true"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time `gorm:"index"`
}

// TableName specifies the table name
func (User) TableName() string {
	return "users"
}

// Session represents an active user session
type Session struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index"`
	RefreshToken string    `gorm:"uniqueIndex;not null"`
	DeviceInfo   string
	IPAddress    string
	UserAgent    string
	ExpiresAt    time.Time `gorm:"not null;index"`
	CreatedAt    time.Time
	RevokedAt    *time.Time
}

// TableName specifies the table name
func (Session) TableName() string {
	return "sessions"
}

// OAuthState represents temporary OAuth state for verification
type OAuthState struct {
	State       string `gorm:"primaryKey"`
	Provider    string `gorm:"not null"`
	RedirectURL string
	CreatedAt   time.Time `gorm:"index"`
	ExpiresAt   time.Time `gorm:"not null;index"`
}

// TableName specifies the table name
func (OAuthState) TableName() string {
	return "oauth_states"
}
