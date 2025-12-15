// Package postgres provides PostgreSQL repository implementations.
package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

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
	// Generate a random API key (prefix sk_)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	rawKey := "sk_" + hex.EncodeToString(bytes)
	keyHash := hashKey(rawKey)
	keyPrefix := rawKey[:10] // sk_ + 8 hex chars

	// Store in database using hashed key
	query := `
		INSERT INTO api_keys (
			id, tenant_id, name, key_hash, key_prefix, scopes, rate_limit, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`

	_, err := r.pool.Exec(ctx, query,
		uuid.New(),
		tenantID,
		"Default API Key",
		keyHash,
		keyPrefix,
		[]string{}, // scopes
		1000,       // default rate limit
	)
	if err != nil {
		return "", fmt.Errorf("failed to store API key: %w", err)
	}

	return rawKey, nil
}

// ValidateAPIKey validates an API key and returns the tenant ID.
func (r *APIKeyRepository) ValidateAPIKey(ctx context.Context, apiKey string) (*uuid.UUID, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("invalid API key: empty")
	}

	keyHash := hashKey(apiKey)

	query := `
		SELECT tenant_id FROM api_keys
		WHERE key_hash = $1
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > $2)
	`

	var tenantID uuid.UUID
	err := r.pool.QueryRow(ctx, query, keyHash, time.Now().UTC()).Scan(&tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid API key: %w", err)
	}

	return &tenantID, nil
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
