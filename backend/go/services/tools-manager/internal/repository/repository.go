package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrConflict = errors.New("conflict")
)

// TxBeginner abstracts pgx pools for testing.
type TxBeginner interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// Repository encapsulates catalog persistence.
type Repository struct {
	pool TxBeginner
}

func New(pool TxBeginner) *Repository {
	return &Repository{pool: pool}
}

// Tool represents a tool plus an optional selected version.
type Tool struct {
	ID           uuid.UUID
	Name         string
	DisplayName  string
	Description  string
	Category     *string
	Tags         []string
	IsPublic     bool
	IsDeprecated bool
	Version      *ToolVersion
}

// ToolVersion captures a specific version of a tool.
type ToolVersion struct {
	ID      uuid.UUID
	Version string
	Status  string
}

// PolicyRule represents an allow/deny rule.
type PolicyRule struct {
	ID      uuid.UUID
	Tenant  *uuid.UUID
	ToolID  *uuid.UUID
	AgentID *string
	Env     *string
	Effect  string
	Scopes  []string
	Roles   []string
	Weight  int
}

// QuotaRule represents rate/usage limits.
type QuotaRule struct {
	ID             uuid.UUID
	Tenant         *uuid.UUID
	ToolID         *uuid.UUID
	AgentID        *string
	Env            *string
	LimitPerMinute *int
	LimitPerDay    *int
	Weight         int
}

// CreateToolParams inputs for creating a tool + version + tenant binding.
type CreateToolParams struct {
	Name         string
	DisplayName  string
	Description  string
	Category     *string
	Tags         []string
	Version      string
	Status       string
	InputSchema  json.RawMessage
	OutputSchema json.RawMessage
	Definition   json.RawMessage
	Allowlist    json.RawMessage
	Metadata     json.RawMessage
	TimeoutSecs  int
	MaxRetries   int
	PayloadLimit int
	IsPublic     bool
	CreatedBy    uuid.UUID
}

// CreatePolicyRuleParams inputs for inserting a policy rule.
type CreatePolicyRuleParams struct {
	TenantID  *uuid.UUID
	ToolID    *uuid.UUID
	AgentID   *string
	Env       *string
	Effect    string
	Scopes    []string
	Roles     []string
	Weight    int
	CreatedBy uuid.UUID
}

// CreateQuotaRuleParams inputs for inserting a quota rule.
type CreateQuotaRuleParams struct {
	TenantID       *uuid.UUID
	ToolID         *uuid.UUID
	AgentID        *string
	Env            *string
	LimitPerMinute *int
	LimitPerDay    *int
	Weight         int
	CreatedBy      uuid.UUID
}

// CreateTool inserts tool + version and binds to tenant with RLS set.
func (r *Repository) CreateTool(ctx context.Context, tenantID string, params CreateToolParams) (*Tool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	var toolID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO tools (name, display_name, description, category, tags, is_public, created_by)
         VALUES ($1,$2,$3,$4,$5,$6,$7)
         RETURNING id`,
		params.Name, params.DisplayName, params.Description, params.Category, params.Tags, params.IsPublic, params.CreatedBy,
	).Scan(&toolID)
	if err != nil {
		if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				return nil, ErrConflict
			}
		}
		return nil, err
	}

	var versionID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO tool_versions (tool_id, version, status, input_schema, output_schema, definition, allowlist, metadata, timeout_seconds, max_retries, created_by)
	 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	 RETURNING id`,
		toolID, params.Version, params.Status, params.InputSchema, params.OutputSchema, params.Definition, params.Allowlist, params.Metadata, params.TimeoutSecs, params.MaxRetries, params.CreatedBy,
	).Scan(&versionID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO tenant_tools (tenant_id, tool_id, tool_version_id, enabled)
         VALUES ($1,$2,$3,TRUE)
         ON CONFLICT (tenant_id, tool_id)
         DO UPDATE SET tool_version_id = EXCLUDED.tool_version_id, enabled = TRUE`,
		tenantID, toolID, versionID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &Tool{
		ID:           toolID,
		Name:         params.Name,
		DisplayName:  params.DisplayName,
		Description:  params.Description,
		Category:     params.Category,
		Tags:         params.Tags,
		IsPublic:     params.IsPublic,
		IsDeprecated: false,
		Version:      &ToolVersion{ID: versionID, Version: params.Version, Status: params.Status},
	}, nil
}

// ListTools returns tools enabled for the tenant.
func (r *Repository) ListTools(ctx context.Context, tenantID string) ([]Tool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT
			t.id, t.name, t.display_name, COALESCE(t.description, ''), t.category, t.tags, t.is_public, t.is_deprecated,
			v.id, v.version, v.status
		FROM tenant_tools tt
		JOIN tools t ON t.id = tt.tool_id
		LEFT JOIN LATERAL (
			SELECT tv.id, tv.version, tv.status
			FROM tool_versions tv
			WHERE tv.tool_id = t.id
			  AND (tt.tool_version_id IS NULL OR tv.id = tt.tool_version_id OR tv.status = 'published')
			ORDER BY (tv.id = tt.tool_version_id) DESC,
					 CASE tv.status WHEN 'published' THEN 0 WHEN 'draft' THEN 1 ELSE 2 END,
					 COALESCE(tv.published_at, tv.created_at) DESC
			LIMIT 1
		) v ON TRUE
		WHERE tt.enabled = TRUE
		ORDER BY t.name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tools []Tool
	for rows.Next() {
		var t Tool
		var category *string
		var versionID *uuid.UUID
		var version, status *string

		if err := rows.Scan(&t.ID, &t.Name, &t.DisplayName, &t.Description, &category, &t.Tags, &t.IsPublic, &t.IsDeprecated, &versionID, &version, &status); err != nil {
			return nil, err
		}
		t.Category = category
		if versionID != nil && version != nil && status != nil {
			t.Version = &ToolVersion{ID: *versionID, Version: *version, Status: *status}
		}
		tools = append(tools, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return tools, nil
}

// CreatePolicyRule inserts an allow/deny rule for a tenant.
func (r *Repository) CreatePolicyRule(ctx context.Context, tenantID string, params CreatePolicyRuleParams) (*PolicyRule, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO policy_rules (tenant_id, tool_id, agent_id, env, effect, scopes, roles, weight, created_by)
	 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	 RETURNING id`,
		params.TenantID, params.ToolID, params.AgentID, params.Env, params.Effect, params.Scopes, params.Roles, params.Weight, params.CreatedBy,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &PolicyRule{
		ID:      id,
		Tenant:  params.TenantID,
		ToolID:  params.ToolID,
		AgentID: params.AgentID,
		Env:     params.Env,
		Effect:  params.Effect,
		Scopes:  params.Scopes,
		Roles:   params.Roles,
		Weight:  params.Weight,
	}, nil
}

// ListPolicyRules returns policy rules visible to the tenant (RLS enforced).
func (r *Repository) ListPolicyRules(ctx context.Context, tenantID string) ([]PolicyRule, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT id, tenant_id, tool_id, agent_id, env, effect, scopes, roles, weight
		FROM policy_rules
		ORDER BY weight DESC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []PolicyRule
	for rows.Next() {
		var rrule PolicyRule
		if err := rows.Scan(&rrule.ID, &rrule.Tenant, &rrule.ToolID, &rrule.AgentID, &rrule.Env, &rrule.Effect, &rrule.Scopes, &rrule.Roles, &rrule.Weight); err != nil {
			return nil, err
		}
		rules = append(rules, rrule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return rules, nil
}

// CreateQuotaRule inserts quota limits for a tenant/tool/agent.
func (r *Repository) CreateQuotaRule(ctx context.Context, tenantID string, params CreateQuotaRuleParams) (*QuotaRule, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO quota_rules (tenant_id, tool_id, agent_id, env, limit_per_minute, limit_per_day, weight, created_by)
	 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	 RETURNING id`,
		params.TenantID, params.ToolID, params.AgentID, params.Env, params.LimitPerMinute, params.LimitPerDay, params.Weight, params.CreatedBy,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &QuotaRule{
		ID:             id,
		Tenant:         params.TenantID,
		ToolID:         params.ToolID,
		AgentID:        params.AgentID,
		Env:            params.Env,
		LimitPerMinute: params.LimitPerMinute,
		LimitPerDay:    params.LimitPerDay,
		Weight:         params.Weight,
	}, nil
}

// ListQuotaRules returns quota rules with RLS enforced.
func (r *Repository) ListQuotaRules(ctx context.Context, tenantID string) ([]QuotaRule, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT id, tenant_id, tool_id, agent_id, env, limit_per_minute, limit_per_day, weight
		FROM quota_rules
		ORDER BY weight DESC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []QuotaRule
	for rows.Next() {
		var rule QuotaRule
		if err := rows.Scan(&rule.ID, &rule.Tenant, &rule.ToolID, &rule.AgentID, &rule.Env, &rule.LimitPerMinute, &rule.LimitPerDay, &rule.Weight); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return rules, nil
}

func setTenant(ctx context.Context, tx pgx.Tx, tenantID string) error {
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE application"); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID)
	return err
}
