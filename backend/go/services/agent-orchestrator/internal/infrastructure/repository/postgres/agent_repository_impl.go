package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/repository"
)

// agentRepositoryImpl implements repository.AgentRepository using PostgreSQL
type agentRepositoryImpl struct {
	db *sql.DB
}

// NewAgentRepository creates a new PostgreSQL agent repository
func NewAgentRepository(db *sql.DB) repository.AgentRepository {
	return &agentRepositoryImpl{
		db: db,
	}
}

// Create creates a new agent
func (r *agentRepositoryImpl) Create(ctx context.Context, agent *entity.Agent) error {
	toolsJSON, err := json.Marshal(agent.Tools)
	if err != nil {
		return fmt.Errorf("failed to marshal tools: %w", err)
	}

	query := `
		INSERT INTO agents (
			id, tenant_id, name, display_name, description,
			system_prompt, model, temperature, max_tokens,
			tools, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.db.ExecContext(
		ctx, query,
		agent.ID, agent.TenantID, agent.Name, agent.DisplayName, agent.Description,
		agent.SystemPrompt, agent.Model, agent.Temperature, agent.MaxTokens,
		toolsJSON, agent.IsActive, agent.CreatedAt, agent.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	return nil
}

// FindByID finds an agent by ID
func (r *agentRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.Agent, error) {
	query := `
		SELECT id, tenant_id, name, display_name, description,
		       system_prompt, model, temperature, max_tokens,
		       tools, is_active, created_at, updated_at
		FROM agents
		WHERE id = $1
	`

	agent := &entity.Agent{}
	var toolsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&agent.ID, &agent.TenantID, &agent.Name, &agent.DisplayName, &agent.Description,
		&agent.SystemPrompt, &agent.Model, &agent.Temperature, &agent.MaxTokens,
		&toolsJSON, &agent.IsActive, &agent.CreatedAt, &agent.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find agent: %w", err)
	}

	if err := json.Unmarshal(toolsJSON, &agent.Tools); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools: %w", err)
	}

	return agent, nil
}

// FindByName finds an agent by name
func (r *agentRepositoryImpl) FindByName(ctx context.Context, tenantID uuid.UUID, name string) (*entity.Agent, error) {
	query := `
		SELECT id, tenant_id, name, display_name, description,
		       system_prompt, model, temperature, max_tokens,
		       tools, is_active, created_at, updated_at
		FROM agents
		WHERE tenant_id = $1 AND name = $2
	`

	agent := &entity.Agent{}
	var toolsJSON []byte

	err := r.db.QueryRowContext(ctx, query, tenantID, name).Scan(
		&agent.ID, &agent.TenantID, &agent.Name, &agent.DisplayName, &agent.Description,
		&agent.SystemPrompt, &agent.Model, &agent.Temperature, &agent.MaxTokens,
		&toolsJSON, &agent.IsActive, &agent.CreatedAt, &agent.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agent not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find agent: %w", err)
	}

	if err := json.Unmarshal(toolsJSON, &agent.Tools); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools: %w", err)
	}

	return agent, nil
}

// FindAll finds all agents
func (r *agentRepositoryImpl) FindAll(ctx context.Context) ([]*entity.Agent, error) {
	query := `
		SELECT id, tenant_id, name, display_name, description,
		       system_prompt, model, temperature, max_tokens,
		       tools, is_active, created_at, updated_at
		FROM agents
		ORDER BY created_at DESC
	`

	return r.queryAgents(ctx, query)
}

// FindByTenant finds all agents for a tenant
func (r *agentRepositoryImpl) FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*entity.Agent, error) {
	query := `
		SELECT id, tenant_id, name, display_name, description,
		       system_prompt, model, temperature, max_tokens,
		       tools, is_active, created_at, updated_at
		FROM agents
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	return r.queryAgents(ctx, query, tenantID)
}

// FindActive finds all active agents
func (r *agentRepositoryImpl) FindActive(ctx context.Context, tenantID uuid.UUID) ([]*entity.Agent, error) {
	query := `
		SELECT id, tenant_id, name, display_name, description,
		       system_prompt, model, temperature, max_tokens,
		       tools, is_active, created_at, updated_at
		FROM agents
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	return r.queryAgents(ctx, query, tenantID)
}

// Update updates an agent
func (r *agentRepositoryImpl) Update(ctx context.Context, agent *entity.Agent) error {
	toolsJSON, err := json.Marshal(agent.Tools)
	if err != nil {
		return fmt.Errorf("failed to marshal tools: %w", err)
	}

	query := `
		UPDATE agents
		SET name = $2, display_name = $3, description = $4,
		    system_prompt = $5, model = $6, temperature = $7, max_tokens = $8,
		    tools = $9, is_active = $10, updated_at = $11
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx, query,
		agent.ID, agent.Name, agent.DisplayName, agent.Description,
		agent.SystemPrompt, agent.Model, agent.Temperature, agent.MaxTokens,
		toolsJSON, agent.IsActive, agent.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	return nil
}

// Delete deletes an agent
func (r *agentRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM agents WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("agent not found: %s", id)
	}

	return nil
}

// Exists checks if an agent with the given name exists
func (r *agentRepositoryImpl) Exists(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM agents WHERE tenant_id = $1 AND name = $2)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, tenantID, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check agent existence: %w", err)
	}

	return exists, nil
}

// Helper methods

func (r *agentRepositoryImpl) queryAgents(ctx context.Context, query string, args ...interface{}) ([]*entity.Agent, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	agents := make([]*entity.Agent, 0)
	for rows.Next() {
		agent := &entity.Agent{}
		var toolsJSON []byte

		err := rows.Scan(
			&agent.ID, &agent.TenantID, &agent.Name, &agent.DisplayName, &agent.Description,
			&agent.SystemPrompt, &agent.Model, &agent.Temperature, &agent.MaxTokens,
			&toolsJSON, &agent.IsActive, &agent.CreatedAt, &agent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}

		if err := json.Unmarshal(toolsJSON, &agent.Tools); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tools: %w", err)
		}

		agents = append(agents, agent)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating agents: %w", err)
	}

	return agents, nil
}
