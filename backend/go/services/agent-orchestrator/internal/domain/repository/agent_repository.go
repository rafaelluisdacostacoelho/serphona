package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
)

// AgentRepository defines the interface for agent persistence
type AgentRepository interface {
	// Create creates a new agent
	Create(ctx context.Context, agent *entity.Agent) error

	// FindByID finds an agent by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Agent, error)

	// FindByName finds an agent by name
	FindByName(ctx context.Context, tenantID uuid.UUID, name string) (*entity.Agent, error)

	// FindAll finds all agents
	FindAll(ctx context.Context) ([]*entity.Agent, error)

	// FindByTenant finds all agents for a tenant
	FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*entity.Agent, error)

	// FindActive finds all active agents
	FindActive(ctx context.Context, tenantID uuid.UUID) ([]*entity.Agent, error)

	// Update updates an agent
	Update(ctx context.Context, agent *entity.Agent) error

	// Delete deletes an agent
	Delete(ctx context.Context, id uuid.UUID) error

	// Exists checks if an agent with the given name exists
	Exists(ctx context.Context, tenantID uuid.UUID, name string) (bool, error)
}
