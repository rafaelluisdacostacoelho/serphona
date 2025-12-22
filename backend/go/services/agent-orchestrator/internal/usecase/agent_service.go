package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/repository"
)

// AgentService defines the interface for agent management
type AgentService interface {
	// CreateAgent creates a new agent
	CreateAgent(ctx context.Context, agent *entity.Agent) error

	// GetAgent retrieves an agent by ID
	GetAgent(ctx context.Context, agentID uuid.UUID) (*entity.Agent, error)

	// GetAgentByName retrieves an agent by name
	GetAgentByName(ctx context.Context, tenantID uuid.UUID, name string) (*entity.Agent, error)

	// ListAgents lists all agents for a tenant
	ListAgents(ctx context.Context, tenantID uuid.UUID, onlyActive bool) ([]*entity.Agent, error)

	// UpdateAgent updates an agent
	UpdateAgent(ctx context.Context, agent *entity.Agent) error

	// DeleteAgent deletes an agent
	DeleteAgent(ctx context.Context, agentID uuid.UUID) error

	// ActivateAgent activates an agent
	ActivateAgent(ctx context.Context, agentID uuid.UUID) error

	// DeactivateAgent deactivates an agent
	DeactivateAgent(ctx context.Context, agentID uuid.UUID) error

	// AddToolToAgent adds a tool to an agent
	AddToolToAgent(ctx context.Context, agentID uuid.UUID, toolName string) error

	// RemoveToolFromAgent removes a tool from an agent
	RemoveToolFromAgent(ctx context.Context, agentID uuid.UUID, toolName string) error
}

// agentServiceImpl implements AgentService
type agentServiceImpl struct {
	agentRepo repository.AgentRepository
}

// NewAgentService creates a new AgentService
func NewAgentService(agentRepo repository.AgentRepository) AgentService {
	return &agentServiceImpl{
		agentRepo: agentRepo,
	}
}

// CreateAgent creates a new agent
func (s *agentServiceImpl) CreateAgent(ctx context.Context, agent *entity.Agent) error {
	// Validate agent
	if err := s.validateAgent(agent); err != nil {
		return fmt.Errorf("invalid agent: %w", err)
	}

	// Check if agent with same name already exists
	exists, err := s.agentRepo.Exists(ctx, agent.TenantID, agent.Name)
	if err != nil {
		return fmt.Errorf("failed to check agent existence: %w", err)
	}
	if exists {
		return fmt.Errorf("agent with name '%s' already exists", agent.Name)
	}

	// Create agent
	if err := s.agentRepo.Create(ctx, agent); err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	return nil
}

// GetAgent retrieves an agent by ID
func (s *agentServiceImpl) GetAgent(ctx context.Context, agentID uuid.UUID) (*entity.Agent, error) {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return agent, nil
}

// GetAgentByName retrieves an agent by name
func (s *agentServiceImpl) GetAgentByName(ctx context.Context, tenantID uuid.UUID, name string) (*entity.Agent, error) {
	agent, err := s.agentRepo.FindByName(ctx, tenantID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent by name: %w", err)
	}

	return agent, nil
}

// ListAgents lists all agents for a tenant
func (s *agentServiceImpl) ListAgents(ctx context.Context, tenantID uuid.UUID, onlyActive bool) ([]*entity.Agent, error) {
	var agents []*entity.Agent
	var err error

	if onlyActive {
		agents, err = s.agentRepo.FindActive(ctx, tenantID)
	} else {
		agents, err = s.agentRepo.FindByTenant(ctx, tenantID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return agents, nil
}

// UpdateAgent updates an agent
func (s *agentServiceImpl) UpdateAgent(ctx context.Context, agent *entity.Agent) error {
	// Validate agent
	if err := s.validateAgent(agent); err != nil {
		return fmt.Errorf("invalid agent: %w", err)
	}

	// Verify agent exists
	existing, err := s.agentRepo.FindByID(ctx, agent.ID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	// Check if name changed and if new name is available
	if existing.Name != agent.Name {
		exists, err := s.agentRepo.Exists(ctx, agent.TenantID, agent.Name)
		if err != nil {
			return fmt.Errorf("failed to check agent existence: %w", err)
		}
		if exists {
			return fmt.Errorf("agent with name '%s' already exists", agent.Name)
		}
	}

	// Update agent
	if err := s.agentRepo.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	return nil
}

// DeleteAgent deletes an agent
func (s *agentServiceImpl) DeleteAgent(ctx context.Context, agentID uuid.UUID) error {
	// Verify agent exists
	if _, err := s.agentRepo.FindByID(ctx, agentID); err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	// Delete agent
	if err := s.agentRepo.Delete(ctx, agentID); err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}

// ActivateAgent activates an agent
func (s *agentServiceImpl) ActivateAgent(ctx context.Context, agentID uuid.UUID) error {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	agent.Activate()

	if err := s.agentRepo.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to activate agent: %w", err)
	}

	return nil
}

// DeactivateAgent deactivates an agent
func (s *agentServiceImpl) DeactivateAgent(ctx context.Context, agentID uuid.UUID) error {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	agent.Deactivate()

	if err := s.agentRepo.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to deactivate agent: %w", err)
	}

	return nil
}

// AddToolToAgent adds a tool to an agent
func (s *agentServiceImpl) AddToolToAgent(ctx context.Context, agentID uuid.UUID, toolName string) error {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	// Check if tool already exists
	if agent.HasTool(toolName) {
		return fmt.Errorf("agent already has tool '%s'", toolName)
	}

	agent.AddTool(toolName)

	if err := s.agentRepo.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to add tool to agent: %w", err)
	}

	return nil
}

// RemoveToolFromAgent removes a tool from an agent
func (s *agentServiceImpl) RemoveToolFromAgent(ctx context.Context, agentID uuid.UUID, toolName string) error {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	// Check if tool exists
	if !agent.HasTool(toolName) {
		return fmt.Errorf("agent does not have tool '%s'", toolName)
	}

	agent.RemoveTool(toolName)

	if err := s.agentRepo.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to remove tool from agent: %w", err)
	}

	return nil
}

// Helper methods

func (s *agentServiceImpl) validateAgent(agent *entity.Agent) error {
	if agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}

	if agent.DisplayName == "" {
		return fmt.Errorf("agent display name is required")
	}

	if agent.SystemPrompt == "" {
		return fmt.Errorf("agent system prompt is required")
	}

	if agent.Model == "" {
		return fmt.Errorf("agent model is required")
	}

	// Validate model
	if !isValidModel(agent.Model) {
		return fmt.Errorf("invalid model: %s", agent.Model)
	}

	// Validate temperature
	if agent.Temperature < 0 || agent.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	// Validate max tokens
	if agent.MaxTokens < 1 {
		return fmt.Errorf("max tokens must be greater than 0")
	}

	return nil
}

func isValidModel(model string) bool {
	validModels := []string{
		entity.ModelGPT4Turbo,
		entity.ModelGPT4,
		entity.ModelGPT35Turbo,
		entity.ModelClaude3Opus,
		entity.ModelClaude3Sonnet,
		entity.ModelClaude3Haiku,
	}

	for _, m := range validModels {
		if model == m {
			return true
		}
	}
	return false
}
