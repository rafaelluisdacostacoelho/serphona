// Package agent provides agent orchestrator client.
package agent

import (
	"bytes"
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

// Client is an HTTP client for agent-orchestrator service.
type Client struct {
	baseURL         string
	httpClient      *http.Client
	logger          *zap.Logger
	serviceToken    string
	serviceName     string
	serviceInstance string
	audience        string
}

// NewClient creates a new agent orchestrator client.
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

// CreateConversationRequest represents a conversation creation request.
type CreateConversationRequest struct {
	TenantID     uuid.UUID              `json:"tenant_id"`
	AgentID      string                 `json:"agent_id"`
	Channel      string                 `json:"channel"` // "voice"
	InitialState map[string]interface{} `json:"initial_state,omitempty"`
}

// ConversationResponse represents a conversation response.
type ConversationResponse struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	AgentID        string    `json:"agent_id"`
	AgentName      string    `json:"agent_name"`
	State          string    `json:"state"`
	CreatedAt      string    `json:"created_at"`
}

// CreateConversation creates a new conversation with an agent.
// POST /api/v1/conversations
func (c *Client) CreateConversation(ctx context.Context, tenantID uuid.UUID, agentID string) (*ConversationResponse, error) {
	req := CreateConversationRequest{
		TenantID: tenantID,
		AgentID:  agentID,
		Channel:  "voice",
		InitialState: map[string]any{
			"call_initiated": time.Now().UTC().Format(time.RFC3339),
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/conversations", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.serviceToken != "" && httpReq.Header.Get("Authorization") == "" {
		if err := validateAudience(c.serviceToken, c.audience); err != nil {
			return nil, err
		}
		httpReq.Header.Set("Authorization", "Bearer "+c.serviceToken)
	}
	c.applyServiceIdentity(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var conversationResp ConversationResponse
	if err := json.NewDecoder(resp.Body).Decode(&conversationResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("conversation created",
		zap.String("conversation_id", conversationResp.ConversationID.String()),
		zap.String("agent_id", agentID),
	)

	return &conversationResp, nil
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

// SubmitTurnRequest represents a conversation turn submission.
type SubmitTurnRequest struct {
	UserMessage string         `json:"user_message"`
	Context     map[string]any `json:"context,omitempty"`
}

// TurnResponse represents an agent's response to a turn.
type TurnResponse struct {
	ConversationID uuid.UUID      `json:"conversation_id"`
	TurnID         uuid.UUID      `json:"turn_id"`
	AgentResponse  string         `json:"agent_response"`
	Intent         string         `json:"intent,omitempty"`
	Action         string         `json:"action,omitempty"`
	ActionParams   map[string]any `json:"action_params,omitempty"`
	State          string         `json:"state"`
	FinishReason   string         `json:"finish_reason,omitempty"`
}

// SubmitTurn submits a user message and gets agent response.
// POST /api/v1/conversations/{conversation_id}/turns
func (c *Client) SubmitTurn(ctx context.Context, conversationID uuid.UUID, userMessage string, context map[string]interface{}) (*TurnResponse, error) {
	req := SubmitTurnRequest{
		UserMessage: userMessage,
		Context:     context,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/conversations/%s/turns", c.baseURL, conversationID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.serviceToken != "" && httpReq.Header.Get("Authorization") == "" {
		if err := validateAudience(c.serviceToken, c.audience); err != nil {
			return nil, err
		}
		httpReq.Header.Set("Authorization", "Bearer "+c.serviceToken)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var turnResp TurnResponse
	if err := json.NewDecoder(resp.Body).Decode(&turnResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("turn submitted",
		zap.String("conversation_id", conversationID.String()),
		zap.String("intent", turnResp.Intent),
		zap.String("action", turnResp.Action),
	)

	return &turnResp, nil
}

// GetAgentResponse gets the current agent for a conversation.
// GET /api/v1/conversations/{conversation_id}/agent
func (c *Client) GetAgentResponse(ctx context.Context, conversationID uuid.UUID) (*ConversationResponse, error) {
	url := fmt.Sprintf("%s/api/v1/conversations/%s/agent", c.baseURL, conversationID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if c.serviceToken != "" && req.Header.Get("Authorization") == "" {
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

	var agentResp ConversationResponse
	if err := json.NewDecoder(resp.Body).Decode(&agentResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &agentResp, nil
}

// EndConversationRequest represents a conversation end request.
type EndConversationRequest struct {
	Reason string `json:"reason,omitempty"`
}

// EndConversation ends a conversation.
// POST /api/v1/conversations/{conversation_id}/end
func (c *Client) EndConversation(ctx context.Context, conversationID uuid.UUID, reason string) error {
	req := EndConversationRequest{
		Reason: reason,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/conversations/%s/end", c.baseURL, conversationID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.serviceToken != "" && httpReq.Header.Get("Authorization") == "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.serviceToken)
	}
	c.applyServiceIdentity(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	c.logger.Info("conversation ended",
		zap.String("conversation_id", conversationID.String()),
		zap.String("reason", reason),
	)

	return nil
}

// UpdateContextRequest represents a context update request.
type UpdateContextRequest struct {
	Context map[string]any `json:"context"`
}

// UpdateContext updates conversation context.
// PATCH /api/v1/conversations/{conversation_id}/context
func (c *Client) UpdateContext(ctx context.Context, conversationID uuid.UUID, contextData map[string]interface{}) error {
	req := UpdateContextRequest{
		Context: contextData,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/conversations/%s/context", c.baseURL, conversationID)
	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.serviceToken != "" && httpReq.Header.Get("Authorization") == "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.serviceToken)
	}
	c.applyServiceIdentity(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
