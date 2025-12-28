package registry

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type poolQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// PostgresLoader loads tools from Postgres for a tenant.
type PostgresLoader struct {
	pool poolQuerier
}

// NewPostgresLoader creates a loader with a pgx pool.
func NewPostgresLoader(pool poolQuerier) *PostgresLoader {
	return &PostgresLoader{pool: pool}
}

// ListTools fetches tools for a tenant with basic fields; assumes RLS or WHERE tenant_id.
func (p *PostgresLoader) ListTools(ctx context.Context, tenantID string) ([]protocol.Tool, error) {
	rows, err := p.pool.Query(ctx, listQuery, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tools := []protocol.Tool{}
	for rows.Next() {
		var t protocol.Tool
		var scopes []string
		var tags []string
		var hosts []string
		var maxDurationMs int64
		if err := rows.Scan(
			&t.Name,
			&t.Version,
			&t.DisplayName,
			&t.Summary,
			&t.Description,
			&t.InputSchema,
			&t.OutputSchema,
			&scopes,
			&tags,
			&hosts,
			&t.IdempotencyKey,
			&t.MaxBodyBytes,
			&maxDurationMs,
			&t.Deprecated,
			&t.ETag,
			&t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		t.Scopes = scopes
		t.Tags = tags
		t.AllowHosts = hosts
		if maxDurationMs > 0 {
			t.MaxDuration = time.Duration(maxDurationMs) * time.Millisecond
		}
		t.TenantID = tenantID
		tools = append(tools, t)
	}
	return tools, rows.Err()
}

// DescribeTool fetches a single tool by name for a tenant.
func (p *PostgresLoader) DescribeTool(ctx context.Context, tenantID, name string) (protocol.Tool, error) {
	row := p.pool.QueryRow(ctx, describeQuery, tenantID, name)

	var t protocol.Tool
	var scopes []string
	var tags []string
	var hosts []string
	var maxDurationMs int64
	if err := row.Scan(
		&t.Name,
		&t.Version,
		&t.DisplayName,
		&t.Summary,
		&t.Description,
		&t.InputSchema,
		&t.OutputSchema,
		&scopes,
		&tags,
		&hosts,
		&t.IdempotencyKey,
		&t.MaxBodyBytes,
		&maxDurationMs,
		&t.Deprecated,
		&t.ETag,
		&t.UpdatedAt,
	); err != nil {
		return protocol.Tool{}, err
	}
	t.Scopes = scopes
	t.Tags = tags
	t.AllowHosts = hosts
	if maxDurationMs > 0 {
		t.MaxDuration = time.Duration(maxDurationMs) * time.Millisecond
	}
	t.TenantID = tenantID
	return t, nil
}

const listQuery = `
SELECT name, version, display_name, summary, description,
       input_schema, output_schema, scopes, tags, allow_hosts,
       idempotency_key, max_body_bytes, max_duration_ms,
       deprecated, etag, updated_at
  FROM tools
 WHERE tenant_id = $1 AND deprecated = false;
`

const describeQuery = `
SELECT name, version, display_name, summary, description,
       input_schema, output_schema, scopes, tags, allow_hosts,
       idempotency_key, max_body_bytes, max_duration_ms,
       deprecated, etag, updated_at
  FROM tools
 WHERE tenant_id = $1 AND name = $2 AND deprecated = false;
`

// RowsToTools is a helper to convert pgx.Rows into []protocol.Tool (usable in tests/mocks).
func RowsToTools(rows pgx.Rows, tenantID string) ([]protocol.Tool, error) {
	defer rows.Close()
	tools := []protocol.Tool{}
	for rows.Next() {
		var t protocol.Tool
		var scopes []string
		var tags []string
		var hosts []string
		var maxDurationMs int64
		if err := rows.Scan(
			&t.Name,
			&t.Version,
			&t.DisplayName,
			&t.Summary,
			&t.Description,
			&t.InputSchema,
			&t.OutputSchema,
			&scopes,
			&tags,
			&hosts,
			&t.IdempotencyKey,
			&t.MaxBodyBytes,
			&maxDurationMs,
			&t.Deprecated,
			&t.ETag,
			&t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		t.Scopes = scopes
		t.Tags = tags
		t.AllowHosts = hosts
		if maxDurationMs > 0 {
			t.MaxDuration = time.Duration(maxDurationMs) * time.Millisecond
		}
		t.TenantID = tenantID
		tools = append(tools, t)
	}
	return tools, rows.Err()
}
