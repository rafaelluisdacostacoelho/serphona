package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
)

// CreateAgentRequest represents the request to create an agent
type CreateAgentRequest struct {
	TenantID     string   `json:"tenant_id" binding:"required"`
	Name         string   `json:"name" binding:"required"`
	DisplayName  string   `json:"display_name" binding:"required"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"system_prompt" binding:"required"`
	Model        string   `json:"model" binding:"required"`
	Temperature  float64  `json:"temperature"`
	MaxTokens    int      `json:"max_tokens"`
	Tools        []string `json:"tools"`
	IsActive     bool     `json:"is_active"`
}

// UpdateAgentRequest represents the request to update an agent
type UpdateAgentRequest struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"system_prompt"`
	Model        string   `json:"model"`
	Temperature  *float64 `json:"temperature"`
	MaxTokens    *int     `json:"max_tokens"`
	Tools        []string `json:"tools"`
	IsActive     *bool    `json:"is_active"`
}

// AgentResponse represents the response for an agent
type AgentResponse struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description"`
	SystemPrompt string    `json:"system_prompt"`
	Model        string    `json:"model"`
	Temperature  float64   `json:"temperature"`
	MaxTokens    int       `json:"max_tokens"`
	Tools        []string  `json:"tools"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToAgentResponse converts an Agent entity to AgentResponse DTO
func ToAgentResponse(agent *entity.Agent) *AgentResponse {
	return &AgentResponse{
		ID:           agent.ID.String(),
		TenantID:     agent.TenantID.String(),
		Name:         agent.Name,
		DisplayName:  agent.DisplayName,
		Description:  agent.Description,
		SystemPrompt: agent.SystemPrompt,
		Model:        agent.Model,
		Temperature:  agent.Temperature,
		MaxTokens:    agent.MaxTokens,
		Tools:        agent.Tools,
		IsActive:     agent.IsActive,
		CreatedAt:    agent.CreatedAt,
		UpdatedAt:    agent.UpdatedAt,
	}
}

// ToAgentResponseList converts a slice of Agent entities to a slice of AgentResponse DTOs
func ToAgentResponseList(agents []*entity.Agent) []*AgentResponse {
	responses := make([]*AgentResponse, len(agents))
	for i, agent := range agents {
		responses[i] = ToAgentResponse(agent)
	}
	return responses
}

// ToAgentEntity converts CreateAgentRequest to Agent entity
func (r *CreateAgentRequest) ToAgentEntity() (*entity.Agent, error) {
	tenantID, err := uuid.Parse(r.TenantID)
	if err != nil {
		return nil, err
	}

	agent := entity.NewAgent(
		tenantID,
		r.Name,
		r.DisplayName,
		r.Description,
		r.SystemPrompt,
	)

	agent.Model = r.Model
	if r.Temperature > 0 {
		agent.Temperature = r.Temperature
	}
	if r.MaxTokens > 0 {
		agent.MaxTokens = r.MaxTokens
	}
	if r.Tools != nil {
		agent.Tools = r.Tools
	}
	agent.IsActive = r.IsActive

	return agent, nil
}
