// Package postgres provides PostgreSQL repository implementations.
package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tenant-manager/internal/domain/apikey"
)

// APIKeyRepository implements apikey.Repository using PostgreSQL.
// Note: persiste apenas campos suportados pelo schema atual (scopes, rate_limit, timestamps, revogação).
type APIKeyRepository struct {
	pool *pgxpool.Pool
}

// NewAPIKeyRepository creates a new APIKeyRepository.
func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

// Save saves a new API key.
func (r *APIKeyRepository) Save(ctx context.Context, key *apikey.APIKey) error {
	query := `
		INSERT INTO api_keys (
			id, tenant_id, name, key_hash, key_prefix, scopes, rate_limit, expires_at, created_at, last_used_at, revoked_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`
	_, err := r.pool.Exec(ctx, query,
		key.ID,
		key.TenantID,
		key.Name,
		key.KeyHash,
		key.KeyPrefix,
		key.Permissions,
		key.Metadata.RateLimit,
		key.ExpiresAt,
		key.CreatedAt,
		key.LastUsedAt,
		key.RevokedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save api key: %w", err)
	}
	return nil
}

// Update updates an existing API key.
func (r *APIKeyRepository) Update(ctx context.Context, key *apikey.APIKey) error {
	query := `
		UPDATE api_keys SET
			name = $2,
			scopes = $3,
			rate_limit = $4,
			expires_at = $5,
			last_used_at = $6,
			revoked_at = $7,
			key_hash = $8,
			key_prefix = $9
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query,
		key.ID,
		key.Name,
		key.Permissions,
		key.Metadata.RateLimit,
		key.ExpiresAt,
		key.LastUsedAt,
		key.RevokedAt,
		key.KeyHash,
		key.KeyPrefix,
	)
	if err != nil {
		return fmt.Errorf("failed to update api key: %w", err)
	}
	return nil
}

// FindByID finds an API key by ID.
func (r *APIKeyRepository) FindByID(ctx context.Context, id uuid.UUID) (*apikey.APIKey, error) {
	query := `
		SELECT id, tenant_id, name, key_hash, key_prefix, scopes, rate_limit, expires_at, last_used_at, created_at, revoked_at
		FROM api_keys
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)
	return scanAPIKey(row)
}

// FindByTenantID finds all API keys for a tenant.
func (r *APIKeyRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*apikey.APIKey, error) {
	query := `
		SELECT id, tenant_id, name, key_hash, key_prefix, scopes, rate_limit, expires_at, last_used_at, created_at, revoked_at
		FROM api_keys
		WHERE tenant_id = $1
	`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query api keys: %w", err)
	}
	defer rows.Close()

	var keys []*apikey.APIKey
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// FindByKeyHash finds an API key by its hash.
func (r *APIKeyRepository) FindByKeyHash(ctx context.Context, keyHash string) (*apikey.APIKey, error) {
	query := `
		SELECT id, tenant_id, name, key_hash, key_prefix, scopes, rate_limit, expires_at, last_used_at, created_at, revoked_at
		FROM api_keys
		WHERE key_hash = $1
	`
	row := r.pool.QueryRow(ctx, query, keyHash)
	return scanAPIKey(row)
}

// FindByKeyPrefix finds an API key by its prefix.
func (r *APIKeyRepository) FindByKeyPrefix(ctx context.Context, prefix string) (*apikey.APIKey, error) {
	query := `
		SELECT id, tenant_id, name, key_hash, key_prefix, scopes, rate_limit, expires_at, last_used_at, created_at, revoked_at
		FROM api_keys
		WHERE key_prefix = $1
	`
	row := r.pool.QueryRow(ctx, query, prefix)
	return scanAPIKey(row)
}

// FindActiveByTenantID finds all active API keys for a tenant.
func (r *APIKeyRepository) FindActiveByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*apikey.APIKey, error) {
	query := `
		SELECT id, tenant_id, name, key_hash, key_prefix, scopes, rate_limit, expires_at, last_used_at, created_at, revoked_at
		FROM api_keys
		WHERE tenant_id = $1
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > $2)
	`
	rows, err := r.pool.Query(ctx, query, tenantID, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query active api keys: %w", err)
	}
	defer rows.Close()

	var keys []*apikey.APIKey
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// Delete deletes an API key (hard delete).
func (r *APIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM api_keys WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete api key: %w", err)
	}
	return nil
}

// ExistsByName checks if an API key with the given name exists for a tenant.
func (r *APIKeyRepository) ExistsByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM api_keys WHERE tenant_id = $1 AND name = $2)`, tenantID, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check api key name: %w", err)
	}
	return exists, nil
}

// CountByTenantID counts API keys for a tenant.
func (r *APIKeyRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM api_keys WHERE tenant_id = $1`, tenantID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count api keys: %w", err)
	}
	return count, nil
}

// CountActiveByTenantID counts active API keys for a tenant.
func (r *APIKeyRepository) CountActiveByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM api_keys
		WHERE tenant_id = $1 AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > $2)
	`, tenantID, time.Now().UTC()).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count active api keys: %w", err)
	}
	return count, nil
}

// ListExpired lists expired API keys (up to limit).
func (r *APIKeyRepository) ListExpired(ctx context.Context, limit int) ([]*apikey.APIKey, error) {
	query := `
		SELECT id, tenant_id, name, key_hash, key_prefix, scopes, rate_limit, expires_at, last_used_at, created_at, revoked_at
		FROM api_keys
		WHERE expires_at IS NOT NULL AND expires_at <= $1 AND revoked_at IS NULL
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, time.Now().UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list expired api keys: %w", err)
	}
	defer rows.Close()

	var keys []*apikey.APIKey
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// RevokeAll revokes all API keys for a tenant.
func (r *APIKeyRepository) RevokeAll(ctx context.Context, tenantID uuid.UUID, revokedBy uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		UPDATE api_keys
		SET revoked_at = $2
		WHERE tenant_id = $1 AND revoked_at IS NULL
	`, tenantID, now)
	if err != nil {
		return fmt.Errorf("failed to revoke all api keys: %w", err)
	}
	return nil
}

// FindByKeyPrefix is required by Repository but already implemented above (duplicate would clash).

// Helper to scan an API key from pgx Row / Rows.
func scanAPIKey(scanner interface {
	Scan(dest ...interface{}) error
}) (*apikey.APIKey, error) {
	var key apikey.APIKey
	var scopes []string
	err := scanner.Scan(
		&key.ID,
		&key.TenantID,
		&key.Name,
		&key.KeyHash,
		&key.KeyPrefix,
		&scopes,
		&key.Metadata.RateLimit,
		&key.ExpiresAt,
		&key.LastUsedAt,
		&key.CreatedAt,
		&key.RevokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("api key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan api key: %w", err)
	}
	key.Permissions = scopes

	// Derive status
	key.Status = apikey.StatusActive
	if key.RevokedAt != nil {
		key.Status = apikey.StatusRevoked
	} else if key.ExpiresAt != nil && time.Now().UTC().After(*key.ExpiresAt) {
		key.Status = apikey.StatusExpired
	}

	return &key, nil
}

// hashKey utility (SHA-256 hex).
func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
