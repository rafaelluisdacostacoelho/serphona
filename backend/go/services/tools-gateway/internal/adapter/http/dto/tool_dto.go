package dto

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

// CreateToolRequest represents the request to create a tool
type CreateToolRequest struct {
	Name               string          `json:"name" binding:"required"`
	DisplayName        string          `json:"display_name" binding:"required"`
	Description        string          `json:"description"`
	Category           string          `json:"category" binding:"required"`
	Method             string          `json:"method" binding:"required"`
	BaseURL            string          `json:"base_url" binding:"required,url"`
	EndpointPath       string          `json:"endpoint_path" binding:"required"`
	Headers            json.RawMessage `json:"headers"`
	AuthType           string          `json:"auth_type" binding:"required"`
	AuthConfig         json.RawMessage `json:"auth_config" binding:"required"`
	InputSchema        json.RawMessage `json:"input_schema" binding:"required"`
	OutputSchema       json.RawMessage `json:"output_schema" binding:"required"`
	TimeoutSeconds     int             `json:"timeout_seconds"`
	MaxRetries         int             `json:"max_retries"`
	RetryDelaySeconds  int             `json:"retry_delay_seconds"`
	RateLimitPerMinute int             `json:"rate_limit_per_minute"`
	RateLimitPerHour   int             `json:"rate_limit_per_hour"`
	CreditCost         int             `json:"credit_cost"`
	IsPublic           bool            `json:"is_public"`
}

// UpdateToolRequest represents the request to update a tool
type UpdateToolRequest struct {
	DisplayName        string          `json:"display_name"`
	Description        string          `json:"description"`
	Category           string          `json:"category"`
	Method             string          `json:"method"`
	BaseURL            string          `json:"base_url"`
	EndpointPath       string          `json:"endpoint_path"`
	Headers            json.RawMessage `json:"headers"`
	AuthType           string          `json:"auth_type"`
	AuthConfig         json.RawMessage `json:"auth_config"`
	InputSchema        json.RawMessage `json:"input_schema"`
	OutputSchema       json.RawMessage `json:"output_schema"`
	TimeoutSeconds     *int            `json:"timeout_seconds"`
	MaxRetries         *int            `json:"max_retries"`
	RetryDelaySeconds  *int            `json:"retry_delay_seconds"`
	RateLimitPerMinute *int            `json:"rate_limit_per_minute"`
	RateLimitPerHour   *int            `json:"rate_limit_per_hour"`
	CreditCost         *int            `json:"credit_cost"`
	IsActive           *bool           `json:"is_active"`
	IsPublic           *bool           `json:"is_public"`
}

// ToolResponse represents a tool response
type ToolResponse struct {
	ID                 uuid.UUID       `json:"id"`
	Name               string          `json:"name"`
	DisplayName        string          `json:"display_name"`
	Description        string          `json:"description"`
	Category           string          `json:"category"`
	Method             string          `json:"method"`
	BaseURL            string          `json:"base_url"`
	EndpointPath       string          `json:"endpoint_path"`
	Headers            json.RawMessage `json:"headers"`
	AuthType           string          `json:"auth_type"`
	InputSchema        json.RawMessage `json:"input_schema"`
	OutputSchema       json.RawMessage `json:"output_schema"`
	TimeoutSeconds     int             `json:"timeout_seconds"`
	MaxRetries         int             `json:"max_retries"`
	RetryDelaySeconds  int             `json:"retry_delay_seconds"`
	RateLimitPerMinute int             `json:"rate_limit_per_minute"`
	RateLimitPerHour   int             `json:"rate_limit_per_hour"`
	CreditCost         int             `json:"credit_cost"`
	IsActive           bool            `json:"is_active"`
	IsPublic           bool            `json:"is_public"`
	CreatedAt          string          `json:"created_at"`
	UpdatedAt          string          `json:"updated_at"`
}

// ListToolsResponse represents the response for listing tools
type ListToolsResponse struct {
	Tools  []*ToolResponse `json:"tools"`
	Total  int64           `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// ToEntity converts CreateToolRequest to entity.Tool
func (r *CreateToolRequest) ToEntity() *entity.Tool {
	tool := &entity.Tool{
		Name:         r.Name,
		DisplayName:  r.DisplayName,
		Description:  r.Description,
		Category:     r.Category,
		Method:       r.Method,
		BaseURL:      r.BaseURL,
		EndpointPath: r.EndpointPath,
		Headers:      r.Headers,
		AuthType:     r.AuthType,
		AuthConfig:   r.AuthConfig,
		InputSchema:  r.InputSchema,
		OutputSchema: r.OutputSchema,
		IsPublic:     r.IsPublic,
		IsActive:     true,
	}

	// Set defaults
	if r.TimeoutSeconds > 0 {
		tool.TimeoutSeconds = r.TimeoutSeconds
	} else {
		tool.TimeoutSeconds = 30
	}

	if r.MaxRetries > 0 {
		tool.MaxRetries = r.MaxRetries
	} else {
		tool.MaxRetries = 3
	}

	if r.RetryDelaySeconds > 0 {
		tool.RetryDelaySeconds = r.RetryDelaySeconds
	} else {
		tool.RetryDelaySeconds = 1
	}

	if r.RateLimitPerMinute > 0 {
		tool.RateLimitPerMinute = r.RateLimitPerMinute
	} else {
		tool.RateLimitPerMinute = 60
	}

	if r.RateLimitPerHour > 0 {
		tool.RateLimitPerHour = r.RateLimitPerHour
	} else {
		tool.RateLimitPerHour = 1000
	}

	if r.CreditCost > 0 {
		tool.CreditCost = r.CreditCost
	} else {
		tool.CreditCost = 1
	}

	return tool
}

// FromEntity converts entity.Tool to ToolResponse
func FromEntity(tool *entity.Tool) *ToolResponse {
	return &ToolResponse{
		ID:                 tool.ID,
		Name:               tool.Name,
		DisplayName:        tool.DisplayName,
		Description:        tool.Description,
		Category:           tool.Category,
		Method:             tool.Method,
		BaseURL:            tool.BaseURL,
		EndpointPath:       tool.EndpointPath,
		Headers:            tool.Headers,
		AuthType:           tool.AuthType,
		InputSchema:        tool.InputSchema,
		OutputSchema:       tool.OutputSchema,
		TimeoutSeconds:     tool.TimeoutSeconds,
		MaxRetries:         tool.MaxRetries,
		RetryDelaySeconds:  tool.RetryDelaySeconds,
		RateLimitPerMinute: tool.RateLimitPerMinute,
		RateLimitPerHour:   tool.RateLimitPerHour,
		CreditCost:         tool.CreditCost,
		IsActive:           tool.IsActive,
		IsPublic:           tool.IsPublic,
		CreatedAt:          tool.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          tool.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
