// Package tenant provides tenant manager client.
package tenant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	authclient "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"
	"go.uber.org/zap"
)

// Client is an HTTP client for tenant-manager service.
type Client struct {
	baseURL         string
	httpClient      *http.Client
	logger          *zap.Logger
	serviceToken    string
	serviceName     string
	serviceInstance string
	audience        string
}

// NewClient creates a new tenant manager client.
func NewClient(baseURL, serviceToken, serviceName, serviceInstance, audience string, logger *zap.Logger) *Client {
	return &Client{
		baseURL:      baseURL,
		serviceToken: serviceToken,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: authclient.WithServiceTransport(nil, serviceName, serviceInstance),
		},
		serviceName:     serviceName,
		serviceInstance: serviceInstance,
		audience:        audience,
		logger:          logger,
	}
}

func (c *Client) applyServiceIdentity(req *http.Request) {
	if req == nil {
		return
	}
	if c.serviceName != "" && req.Header.Get("X-Service-Name") == "" {
		req.Header.Set("X-Service-Name", c.serviceName)
	}
	if c.serviceInstance != "" && req.Header.Get("X-Service-Instance") == "" {
		req.Header.Set("X-Service-Instance", c.serviceInstance)
	}
}

func validateAudience(token, expected string) error {
	if expected == "" {
		return nil
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims := &jwt.RegisteredClaims{}

	if _, _, err := parser.ParseUnverified(token, claims); err != nil {
		return err
	}

	for _, aud := range claims.Audience {
		if aud == expected {
			return nil
		}
	}

	return fmt.Errorf("service token audience mismatch")
}

// DIDInfo represents DID lookup information.
type DIDInfo struct {
	DID      string    `json:"did"`
	TenantID uuid.UUID `json:"tenant_id"`
	Enabled  bool      `json:"enabled"`
}

// LookupDID looks up a DID to find the associated tenant.
// GET /api/v1/telephony/dids/lookup/{phone_number}
func (c *Client) LookupDID(ctx context.Context, phoneNumber string) (*DIDInfo, error) {
	url := fmt.Sprintf("%s/api/v1/telephony/dids/lookup/%s", c.baseURL, phoneNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if c.serviceToken != "" && req.Header.Get("Authorization") == "" {
		if err := validateAudience(c.serviceToken, c.audience); err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.serviceToken)
	}
	c.applyServiceIdentity(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("DID not found: %s", phoneNumber)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var didInfo DIDInfo
	if err := json.NewDecoder(resp.Body).Decode(&didInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("DID lookup successful",
		zap.String("phone_number", phoneNumber),
		zap.String("tenant_id", didInfo.TenantID.String()),
	)

	return &didInfo, nil
}

// ProviderSettings represents STT/TTS/LLM provider configuration.
type ProviderSettings struct {
	STTProvider string                 `json:"stt_provider"` // "google", "azure", etc
	TTSProvider string                 `json:"tts_provider"` // "google", "elevenlabs", etc
	LLMProvider string                 `json:"llm_provider"` // "openai", "anthropic", etc
	STTConfig   map[string]interface{} `json:"stt_config"`
	TTSConfig   map[string]interface{} `json:"tts_config"`
	LLMConfig   map[string]interface{} `json:"llm_config"`
}

// GetProviderSettings retrieves provider settings for a tenant.
// GET /api/v1/tenants/{tenant_id}/telephony/provider-settings
func (c *Client) GetProviderSettings(ctx context.Context, tenantID uuid.UUID) (*ProviderSettings, error) {
	url := fmt.Sprintf("%s/api/v1/tenants/%s/telephony/provider-settings", c.baseURL, tenantID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if c.serviceToken != "" && req.Header.Get("Authorization") == "" {
		if err := validateAudience(c.serviceToken, c.audience); err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.serviceToken)
	}
	c.applyServiceIdentity(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var settings ProviderSettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("provider settings retrieved",
		zap.String("tenant_id", tenantID.String()),
		zap.String("stt_provider", settings.STTProvider),
		zap.String("tts_provider", settings.TTSProvider),
	)

	return &settings, nil
}

// AgentConfig represents agent configuration from prompts.yaml.
type AgentConfig struct {
	AgentID          string                 `json:"agent_id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	SystemPrompt     string                 `json:"system_prompt"`
	Voice            VoiceConfig            `json:"voice"`
	Routing          RoutingConfig          `json:"routing"`
	Safety           SafetyConfig           `json:"safety"`
	ConversationFlow ConversationFlowConfig `json:"conversation_flow"`
}

// VoiceConfig represents voice configuration.
type VoiceConfig struct {
	Provider string  `json:"provider"`
	VoiceID  string  `json:"voice_id"`
	Rate     float64 `json:"rate"`
	Pitch    float64 `json:"pitch"`
	Language string  `json:"language"`
}

// RoutingConfig represents routing configuration.
type RoutingConfig struct {
	CanRoute          bool     `json:"can_route"`
	AllowedTargets    []string `json:"allowed_targets"`
	TransferIntents   []string `json:"transfer_intents"`
	EscalationTrigger string   `json:"escalation_trigger"`
}

// SafetyConfig represents safety configuration.
type SafetyConfig struct {
	ForbiddenTopics   []string `json:"forbidden_topics"`
	MaxTurns          int      `json:"max_turns"`
	InactivityTimeout int      `json:"inactivity_timeout_seconds"`
}

// ConversationFlowConfig represents conversation flow configuration.
type ConversationFlowConfig struct {
	MaxRetries        int      `json:"max_retries"`
	ConfirmationSteps []string `json:"confirmation_steps"`
	Handoff           bool     `json:"handoff_enabled"`
}

// GetAgentConfig retrieves agent configuration for a tenant.
// GET /api/v1/tenants/{tenant_id}/agent-config
func (c *Client) GetAgentConfig(ctx context.Context, tenantID uuid.UUID) (*AgentConfig, error) {
	url := fmt.Sprintf("%s/api/v1/tenants/%s/agent-config", c.baseURL, tenantID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if c.serviceToken != "" && req.Header.Get("Authorization") == "" {
		if err := validateAudience(c.serviceToken, c.audience); err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.serviceToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var config AgentConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("agent config retrieved",
		zap.String("tenant_id", tenantID.String()),
		zap.String("agent_id", config.AgentID),
		zap.String("agent_name", config.Name),
	)

	return &config, nil
}

// GetTenantInfo retrieves basic tenant information.
// GET /api/v1/tenants/{tenant_id}
func (c *Client) GetTenantInfo(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/tenants/%s", c.baseURL, tenantID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if c.serviceToken != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+c.serviceToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tenantInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tenantInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tenantInfo, nil
}
