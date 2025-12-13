// Package entity contains domain entities for tools gateway.
package entity

import (
	"time"

	"github.com/google/uuid"
)

// IntegrationType represents the type of external integration.
type IntegrationType string

const (
	IntegrationTypeREST      IntegrationType = "rest"
	IntegrationTypeGraphQL   IntegrationType = "graphql"
	IntegrationTypeSOAP      IntegrationType = "soap"
	IntegrationTypeGRPC      IntegrationType = "grpc"
	IntegrationTypeWebhook   IntegrationType = "webhook"
	IntegrationTypeWebSocket IntegrationType = "websocket"
)

// OAuth2GrantType represents OAuth 2.0 grant types.
type OAuth2GrantType string

const (
	GrantTypeClientCredentials OAuth2GrantType = "client_credentials"
	GrantTypeAuthorizationCode OAuth2GrantType = "authorization_code"
	GrantTypeRefreshToken      OAuth2GrantType = "refresh_token"
	GrantTypePassword          OAuth2GrantType = "password"
	GrantTypeImplicit          OAuth2GrantType = "implicit"
)

// Integration represents an external API integration configuration.
type Integration struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primary_key"`
	TenantID    uuid.UUID       `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name        string          `json:"name" gorm:"type:varchar(255);not null"`
	DisplayName string          `json:"display_name" gorm:"type:varchar(255);not null"`
	Description string          `json:"description" gorm:"type:text"`
	Provider    string          `json:"provider" gorm:"type:varchar(100);index"` // e.g., "google", "salesforce", "custom"
	Type        IntegrationType `json:"type" gorm:"type:varchar(50);not null"`
	BaseURL     string          `json:"base_url" gorm:"type:text;not null"`

	// Authentication
	AuthType   AuthType               `json:"auth_type" gorm:"type:varchar(50);not null"`
	AuthConfig map[string]interface{} `json:"auth_config" gorm:"type:jsonb"`

	// OAuth 2.0 specific
	OAuth2Config *OAuth2Config `json:"oauth2_config,omitempty" gorm:"type:jsonb"`

	// Protocol specific configs
	GraphQLConfig *GraphQLConfig `json:"graphql_config,omitempty" gorm:"type:jsonb"`
	SOAPConfig    *SOAPConfig    `json:"soap_config,omitempty" gorm:"type:jsonb"`
	GRPCConfig    *GRPCConfig    `json:"grpc_config,omitempty" gorm:"type:jsonb"`

	// Headers and settings
	DefaultHeaders map[string]string      `json:"default_headers" gorm:"type:jsonb"`
	Settings       map[string]interface{} `json:"settings" gorm:"type:jsonb"`

	// Status
	IsActive bool                   `json:"is_active" gorm:"default:true"`
	Metadata map[string]interface{} `json:"metadata" gorm:"type:jsonb"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OAuth2Config holds OAuth 2.0 configuration.
type OAuth2Config struct {
	ClientID     string          `json:"client_id"`
	ClientSecret string          `json:"client_secret"` // Encrypted in DB
	AuthURL      string          `json:"auth_url"`
	TokenURL     string          `json:"token_url"`
	RedirectURL  string          `json:"redirect_url"`
	Scopes       []string        `json:"scopes"`
	GrantType    OAuth2GrantType `json:"grant_type"`

	// Advanced
	TokenEndpointAuthMethod string            `json:"token_endpoint_auth_method,omitempty"` // client_secret_post, client_secret_basic
	AdditionalParams        map[string]string `json:"additional_params,omitempty"`
}

// GraphQLConfig holds GraphQL-specific configuration.
type GraphQLConfig struct {
	Endpoint         string                 `json:"endpoint"`
	IntrospectionURL string                 `json:"introspection_url,omitempty"`
	SubscriptionsURL string                 `json:"subscriptions_url,omitempty"`
	BatchingEnabled  bool                   `json:"batching_enabled"`
	DefaultVariables map[string]interface{} `json:"default_variables,omitempty"`
}

// SOAPConfig holds SOAP/WebService configuration.
type SOAPConfig struct {
	WSDLURL     string `json:"wsdl_url"`
	Namespace   string `json:"namespace"`
	ServiceName string `json:"service_name"`
	PortName    string `json:"port_name,omitempty"`
	SOAPVersion string `json:"soap_version"`       // 1.1 or 1.2
	Envelope    string `json:"envelope,omitempty"` // Custom envelope template
}

// GRPCConfig holds gRPC-specific configuration.
type GRPCConfig struct {
	ProtoFile      string            `json:"proto_file,omitempty"`       // Proto file content or URL
	ServiceName    string            `json:"service_name"`               // Service name from proto
	UseTLS         bool              `json:"use_tls"`                    // Enable TLS
	ServerName     string            `json:"server_name,omitempty"`      // For TLS verification
	CertFile       string            `json:"cert_file,omitempty"`        // Client cert
	Reflection     bool              `json:"reflection"`                 // Use server reflection
	Metadata       map[string]string `json:"metadata,omitempty"`         // Default metadata (headers)
	MaxMessageSize int               `json:"max_message_size,omitempty"` // Max message size in bytes
	Timeout        int               `json:"timeout,omitempty"`          // Request timeout in seconds
	KeepAlive      bool              `json:"keep_alive"`                 // Enable keep-alive
	Compression    string            `json:"compression,omitempty"`      // gzip, etc
}

// TableName specifies the table name for GORM.
func (Integration) TableName() string {
	return "integrations"
}

// BeforeCreate hook for GORM.
func (i *Integration) BeforeCreate() error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	now := time.Now().UTC()
	i.CreatedAt = now
	i.UpdatedAt = now
	return nil
}

// BeforeUpdate hook for GORM.
func (i *Integration) BeforeUpdate() error {
	i.UpdatedAt = time.Now().UTC()
	return nil
}
