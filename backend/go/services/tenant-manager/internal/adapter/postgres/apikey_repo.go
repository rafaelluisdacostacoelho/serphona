// Package postgres provides PostgreSQL repository implementations.
package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// APIKeyRepository implements API key repository using PostgreSQL.
type APIKeyRepository struct {
	pool *pgxpool.Pool
}

// NewAPIKeyRepository creates a new APIKeyRepository.
func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

// GenerateAPIKey generates a new API key for a tenant.
func (r *APIKeyRepository) GenerateAPIKey(ctx context.Context, tenantID uuid.UUID) (string, error) {
	// Generate a random API key
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	apiKey := "sk_" + hex.EncodeToString(bytes)

	// Store in database (simplified - in production, store hash instead)
	query := `
		INSERT INTO api_keys (id, tenant_id, key, name, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := r.pool.Exec(ctx, query, uuid.New(), tenantID, apiKey, "Default API Key")
	if err != nil {
		return "", fmt.Errorf("failed to store API key: %w", err)
	}

	return apiKey, nil
}

// ValidateAPIKey validates an API key and returns the tenant ID.
func (r *APIKeyRepository) ValidateAPIKey(ctx context.Context, apiKey string) (*uuid.UUID, error) {
	query := `
		SELECT tenant_id FROM api_keys
		WHERE key = $1 AND deleted_at IS NULL
	`

	var tenantID uuid.UUID
	err := r.pool.QueryRow(ctx, query, apiKey).Scan(&tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid API key: %w", err)
	}

	return &tenantID, nil
}
