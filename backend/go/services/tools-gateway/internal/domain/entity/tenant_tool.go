package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TenantTool represents a tool configuration specific to a tenant
type TenantTool struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index:idx_tenant_tools_tenant_tool,unique"`
	ToolID   uuid.UUID `json:"tool_id" gorm:"type:uuid;not null;index:idx_tenant_tools_tenant_tool,unique"`

	// Override configurations
	CustomAuthConfig         json.RawMessage `json:"custom_auth_config,omitempty" gorm:"type:jsonb"`
	CustomRateLimitPerMinute *int            `json:"custom_rate_limit_per_minute,omitempty"`
	CustomCreditCost         *int            `json:"custom_credit_cost,omitempty"`

	// Permissions
	IsEnabled      bool        `json:"is_enabled" gorm:"default:true"`
	AllowedUserIDs []uuid.UUID `json:"allowed_user_ids,omitempty" gorm:"type:uuid[]"` // if set, only these users can use

	// Metadata
	CreatedAt time.Time `json:"created_at" gorm:"default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"default:now()"`

	// Relations
	Tool *Tool `json:"tool,omitempty" gorm:"foreignKey:ToolID"`
}

// TableName specifies the table name for GORM
func (TenantTool) TableName() string {
	return "tenant_tools"
}

// GetRateLimitPerMinute returns the effective rate limit (custom or default)
func (tt *TenantTool) GetRateLimitPerMinute(defaultLimit int) int {
	if tt.CustomRateLimitPerMinute != nil {
		return *tt.CustomRateLimitPerMinute
	}
	return defaultLimit
}

// GetCreditCost returns the effective credit cost (custom or default)
func (tt *TenantTool) GetCreditCost(defaultCost int) int {
	if tt.CustomCreditCost != nil {
		return *tt.CustomCreditCost
	}
	return defaultCost
}

// IsUserAllowed checks if a user is allowed to use this tool
func (tt *TenantTool) IsUserAllowed(userID uuid.UUID) bool {
	// If no specific users are set, all users in the tenant can use it
	if len(tt.AllowedUserIDs) == 0 {
		return true
	}

	// Check if user is in the allowed list
	for _, allowedID := range tt.AllowedUserIDs {
		if allowedID == userID {
			return true
		}
	}

	return false
}
