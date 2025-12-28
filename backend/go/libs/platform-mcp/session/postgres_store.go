package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type poolExecQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// PostgresStore persists sessions in Postgres.
type PostgresStore struct {
	pool poolExecQuerier
}

// NewPostgresStore builds a store backed by Postgres.
func NewPostgresStore(pool poolExecQuerier) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (p *PostgresStore) Create(ctx context.Context, s Session) (Session, error) {
	if s.ID == "" || s.TenantID == "" {
		return Session{}, errRequired
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	attrs, _ := json.Marshal(s.Attrs)
	_, err := p.pool.Exec(ctx, `
INSERT INTO mcp_sessions (id, tenant_id, created_at, expires_at, attrs)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (id, tenant_id) DO UPDATE SET expires_at = EXCLUDED.expires_at, attrs = EXCLUDED.attrs;`,
		s.ID, s.TenantID, s.CreatedAt, s.ExpiresAt, attrs)
	if err != nil {
		return Session{}, err
	}
	return s, nil
}

func (p *PostgresStore) Get(ctx context.Context, tenantID, id string) (Session, error) {
	var s Session
	var attrs []byte
	if err := p.pool.QueryRow(ctx, `
SELECT id, tenant_id, created_at, expires_at, attrs FROM mcp_sessions WHERE tenant_id=$1 AND id=$2;`, tenantID, id).
		Scan(&s.ID, &s.TenantID, &s.CreatedAt, &s.ExpiresAt, &attrs); err != nil {
		return Session{}, err
	}
	_ = json.Unmarshal(attrs, &s.Attrs)
	if !s.ExpiresAt.IsZero() && time.Now().After(s.ExpiresAt) {
		_ = p.End(ctx, tenantID, id)
		return Session{}, errExpired
	}
	return s, nil
}

func (p *PostgresStore) Touch(ctx context.Context, tenantID, id string, extend time.Duration) (Session, error) {
	if extend <= 0 {
		return Session{}, errExtend
	}
	var expires time.Time
	if err := p.pool.QueryRow(ctx, `
UPDATE mcp_sessions SET expires_at = COALESCE(expires_at, NOW()) + ($3 * INTERVAL '1 second')
 WHERE tenant_id=$1 AND id=$2
 RETURNING expires_at;`, tenantID, id, extend.Seconds()).Scan(&expires); err != nil {
		return Session{}, err
	}
	// re-read to return full session
	return p.Get(ctx, tenantID, id)
}

func (p *PostgresStore) End(ctx context.Context, tenantID, id string) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM mcp_sessions WHERE tenant_id=$1 AND id=$2;`, tenantID, id)
	return err
}

var (
	errRequired = fmt.Errorf("session id and tenant_id are required")
	errExtend   = fmt.Errorf("extend duration must be positive")
	errExpired  = fmt.Errorf("session expired")
)
