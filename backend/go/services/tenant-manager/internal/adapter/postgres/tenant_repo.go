// Package postgres provides PostgreSQL repository implementations.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tenant-manager/internal/domain/tenant"
)

// TenantRepository implements tenant.Repository using PostgreSQL.
type TenantRepository struct {
	pool *pgxpool.Pool
}

// NewTenantRepository creates a new TenantRepository.
func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

// Create persists a new tenant.
func (r *TenantRepository) Create(ctx context.Context, t *tenant.Tenant) error {
	settingsJSON, err := json.Marshal(t.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	metadataJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO tenants (
			id, name, slug, email, phone, status, plan,
			settings, metadata, stripe_id, billing_email,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13
		)
	`

	_, err = r.pool.Exec(ctx, query,
		t.ID,
		t.Name,
		t.Slug,
		t.Email,
		t.Phone,
		string(t.Status),
		string(t.Plan),
		settingsJSON,
		metadataJSON,
		t.StripeID,
		t.BillingEmail,
		t.CreatedAt,
		t.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			if strings.Contains(err.Error(), "email") {
				return fmt.Errorf("tenant with email %s already exists", t.Email)
			}
			if strings.Contains(err.Error(), "slug") {
				return fmt.Errorf("tenant with slug %s already exists", t.Slug)
			}
		}
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	return nil
}

// GetByID retrieves a tenant by its ID.
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*tenant.Tenant, error) {
	query := `
		SELECT 
			id, name, slug, email, phone, status, plan,
			settings, metadata, stripe_id, billing_email,
			created_at, updated_at, deleted_at
		FROM tenants
		WHERE id = $1 AND deleted_at IS NULL
	`

	return r.scanTenant(ctx, r.pool.QueryRow(ctx, query, id))
}

// GetBySlug retrieves a tenant by its slug.
func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*tenant.Tenant, error) {
	query := `
		SELECT 
			id, name, slug, email, phone, status, plan,
			settings, metadata, stripe_id, billing_email,
			created_at, updated_at, deleted_at
		FROM tenants
		WHERE slug = $1 AND deleted_at IS NULL
	`

	return r.scanTenant(ctx, r.pool.QueryRow(ctx, query, slug))
}

// GetByEmail retrieves a tenant by its email.
func (r *TenantRepository) GetByEmail(ctx context.Context, email string) (*tenant.Tenant, error) {
	query := `
		SELECT 
			id, name, slug, email, phone, status, plan,
			settings, metadata, stripe_id, billing_email,
			created_at, updated_at, deleted_at
		FROM tenants
		WHERE email = $1 AND deleted_at IS NULL
	`

	return r.scanTenant(ctx, r.pool.QueryRow(ctx, query, email))
}

// Update updates an existing tenant.
func (r *TenantRepository) Update(ctx context.Context, t *tenant.Tenant) error {
	settingsJSON, err := json.Marshal(t.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	metadataJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE tenants SET
			name = $2,
			email = $3,
			phone = $4,
			status = $5,
			plan = $6,
			settings = $7,
			metadata = $8,
			stripe_id = $9,
			billing_email = $10,
			updated_at = $11
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query,
		t.ID,
		t.Name,
		t.Email,
		t.Phone,
		string(t.Status),
		string(t.Plan),
		settingsJSON,
		metadataJSON,
		t.StripeID,
		t.BillingEmail,
		time.Now().UTC(),
	)

	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("tenant not found")
	}

	return nil
}

// Delete soft-deletes a tenant.
func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE tenants SET
			status = $2,
			deleted_at = $3,
			updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	now := time.Now().UTC()
	result, err := r.pool.Exec(ctx, query, id, string(tenant.StatusDeleted), now)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("tenant not found")
	}

	return nil
}

// List retrieves tenants with pagination and filtering.
func (r *TenantRepository) List(ctx context.Context, filter tenant.ListFilter) (*tenant.ListResult, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, string(*filter.Status))
		argIndex++
	}

	if filter.Plan != nil {
		conditions = append(conditions, fmt.Sprintf("plan = $%d", argIndex))
		args = append(args, string(*filter.Plan))
		argIndex++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR email ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tenants WHERE %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count tenants: %w", err)
	}

	// Build ORDER BY
	sortBy := "created_at"
	if filter.SortBy != "" {
		// Validate sort column to prevent SQL injection
		validSortColumns := map[string]bool{
			"name": true, "email": true, "created_at": true, "updated_at": true, "status": true,
		}
		if validSortColumns[filter.SortBy] {
			sortBy = filter.SortBy
		}
	}

	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	// Calculate pagination
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageNumber <= 0 {
		filter.PageNumber = 1
	}
	offset := (filter.PageNumber - 1) * filter.PageSize

	// Fetch tenants
	query := fmt.Sprintf(`
		SELECT 
			id, name, slug, email, phone, status, plan,
			settings, metadata, stripe_id, billing_email,
			created_at, updated_at, deleted_at
		FROM tenants
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argIndex, argIndex+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*tenant.Tenant
	for rows.Next() {
		t, err := r.scanTenantFromRows(rows)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenants: %w", err)
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &tenant.ListResult{
		Tenants:    tenants,
		Total:      total,
		PageSize:   filter.PageSize,
		PageNumber: filter.PageNumber,
		TotalPages: totalPages,
	}, nil
}

// UpdateSettings updates only the tenant settings.
func (r *TenantRepository) UpdateSettings(ctx context.Context, id uuid.UUID, settings tenant.Settings) error {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `
		UPDATE tenants SET
			settings = $2,
			updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, id, settingsJSON, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("tenant not found")
	}

	return nil
}

// GetQuota retrieves the quota for a tenant.
func (r *TenantRepository) GetQuota(ctx context.Context, tenantID uuid.UUID) (*tenant.Quota, error) {
	query := `
		SELECT 
			tenant_id, max_api_keys, max_users, max_calls_per_month,
			max_minutes_per_month, max_storage_gb, used_calls,
			used_minutes, used_storage_gb, reset_at
		FROM tenant_quotas
		WHERE tenant_id = $1
	`

	var q tenant.Quota
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(
		&q.TenantID,
		&q.MaxAPIKeys,
		&q.MaxUsers,
		&q.MaxCallsPerMonth,
		&q.MaxMinutesPerMonth,
		&q.MaxStorageGB,
		&q.UsedCalls,
		&q.UsedMinutes,
		&q.UsedStorageGB,
		&q.ResetAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("quota not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get quota: %w", err)
	}

	return &q, nil
}

// UpdateQuota updates the quota for a tenant.
func (r *TenantRepository) UpdateQuota(ctx context.Context, quota *tenant.Quota) error {
	query := `
		INSERT INTO tenant_quotas (
			tenant_id, max_api_keys, max_users, max_calls_per_month,
			max_minutes_per_month, max_storage_gb, used_calls,
			used_minutes, used_storage_gb, reset_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tenant_id) DO UPDATE SET
			max_api_keys = $2,
			max_users = $3,
			max_calls_per_month = $4,
			max_minutes_per_month = $5,
			max_storage_gb = $6,
			used_calls = $7,
			used_minutes = $8,
			used_storage_gb = $9,
			reset_at = $10
	`

	_, err := r.pool.Exec(ctx, query,
		quota.TenantID,
		quota.MaxAPIKeys,
		quota.MaxUsers,
		quota.MaxCallsPerMonth,
		quota.MaxMinutesPerMonth,
		quota.MaxStorageGB,
		quota.UsedCalls,
		quota.UsedMinutes,
		quota.UsedStorageGB,
		quota.ResetAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update quota: %w", err)
	}

	return nil
}

// IncrementUsage increments usage counters for a tenant.
func (r *TenantRepository) IncrementUsage(ctx context.Context, tenantID uuid.UUID, calls, minutes int) error {
	query := `
		UPDATE tenant_quotas SET
			used_calls = used_calls + $2,
			used_minutes = used_minutes + $3
		WHERE tenant_id = $1
	`

	_, err := r.pool.Exec(ctx, query, tenantID, calls, minutes)
	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}

	return nil
}

// ExistsBySlug checks if a tenant with the given slug exists.
func (r *TenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check slug existence: %w", err)
	}

	return exists, nil
}

// ExistsByEmail checks if a tenant with the given email exists.
func (r *TenantRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE email = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return exists, nil
}

// scanTenant scans a single tenant from a row.
func (r *TenantRepository) scanTenant(ctx context.Context, row pgx.Row) (*tenant.Tenant, error) {
	var t tenant.Tenant
	var status, plan string
	var settingsJSON, metadataJSON []byte

	err := row.Scan(
		&t.ID,
		&t.Name,
		&t.Slug,
		&t.Email,
		&t.Phone,
		&status,
		&plan,
		&settingsJSON,
		&metadataJSON,
		&t.StripeID,
		&t.BillingEmail,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.DeletedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("tenant not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan tenant: %w", err)
	}

	t.Status = tenant.Status(status)
	t.Plan = tenant.Plan(plan)

	if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &t.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &t, nil
}

// scanTenantFromRows scans a tenant from rows.
func (r *TenantRepository) scanTenantFromRows(rows pgx.Rows) (*tenant.Tenant, error) {
	var t tenant.Tenant
	var status, plan string
	var settingsJSON, metadataJSON []byte

	err := rows.Scan(
		&t.ID,
		&t.Name,
		&t.Slug,
		&t.Email,
		&t.Phone,
		&status,
		&plan,
		&settingsJSON,
		&metadataJSON,
		&t.StripeID,
		&t.BillingEmail,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan tenant: %w", err)
	}

	t.Status = tenant.Status(status)
	t.Plan = tenant.Plan(plan)

	if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &t.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &t, nil
}
