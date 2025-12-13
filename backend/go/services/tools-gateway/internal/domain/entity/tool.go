package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Tool represents an external API integration
type Tool struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	DisplayName string    `json:"display_name" gorm:"not null"`
	Description string    `json:"description"`
	Category    string    `json:"category" gorm:"index"`

	// HTTP Configuration
	Method       string          `json:"method" gorm:"not null"` // GET, POST, PUT, DELETE, PATCH
	BaseURL      string          `json:"base_url" gorm:"not null"`
	EndpointPath string          `json:"endpoint_path" gorm:"not null"`
	Headers      json.RawMessage `json:"headers" gorm:"type:jsonb;default:'{}'"`

	// Authentication
	AuthType   string          `json:"auth_type" gorm:"not null"` // none, api_key, oauth2, bearer, basic
	AuthConfig json.RawMessage `json:"auth_config" gorm:"type:jsonb;default:'{}'"`

	// Schemas
	InputSchema  json.RawMessage `json:"input_schema" gorm:"type:jsonb;not null"`
	OutputSchema json.RawMessage `json:"output_schema" gorm:"type:jsonb;not null"`

	// Configuration
	TimeoutSeconds    int `json:"timeout_seconds" gorm:"default:30"`
	MaxRetries        int `json:"max_retries" gorm:"default:3"`
	RetryDelaySeconds int `json:"retry_delay_seconds" gorm:"default:1"`

	// Rate Limiting
	RateLimitPerMinute int `json:"rate_limit_per_minute" gorm:"default:60"`
	RateLimitPerHour   int `json:"rate_limit_per_hour" gorm:"default:1000"`

	// Billing
	CreditCost int `json:"credit_cost" gorm:"default:1"`

	// Metadata
	IsActive  bool       `json:"is_active" gorm:"default:true;index"`
	IsPublic  bool       `json:"is_public" gorm:"default:false;index"`
	CreatedAt time.Time  `json:"created_at" gorm:"default:now()"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"default:now()"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
}

// TableName specifies the table name for GORM
func (Tool) TableName() string {
	return "tools"
}

// HTTPMethod constants
const (
	HTTPMethodGET    = "GET"
	HTTPMethodPOST   = "POST"
	HTTPMethodPUT    = "PUT"
	HTTPMethodDELETE = "DELETE"
	HTTPMethodPATCH  = "PATCH"
)

// Category constants
const (
	CategorySearch        = "search"
	CategoryWeather       = "weather"
	CategoryCommunication = "communication"
	CategoryCalendar      = "calendar"
	CategoryDatabase      = "database"
	CategoryFile          = "file"
	CategoryAnalytics     = "analytics"
	CategoryOther         = "other"
)
