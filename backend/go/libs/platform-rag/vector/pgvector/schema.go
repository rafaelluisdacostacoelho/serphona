package pgvector

import (
	"context"
	"database/sql"
	"fmt"
)

const defaultLists = 100

// CreateSchema creates the pgvector table/indexes if they do not exist.
// This is best-effort and should be run in a migration/init step.
func CreateSchema(ctx context.Context, db *sql.DB, cfg Config) error {
	table, err := Store{config: cfg}.tableName()
	if err != nil {
		return err
	}

	dim := cfg.Dimension
	if dim <= 0 {
		dim = 1536
	}
	lists := cfg.Lists
	if lists <= 0 {
		lists = defaultLists
	}

	stmts := []string{
		"CREATE EXTENSION IF NOT EXISTS vector",
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
  tenant_id text NOT NULL,
  namespace text NOT NULL,
  document_id text NOT NULL,
  chunk_id text NOT NULL,
  content text NOT NULL,
  metadata jsonb,
  embedding vector(%d) NOT NULL,
  etag text,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, namespace, chunk_id)
);`, table, dim),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s_embedding_idx ON %s USING ivfflat (embedding vector_l2_ops) WITH (lists = %d);`, table, table, lists),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s_tenant_ns_doc_idx ON %s (tenant_id, namespace, document_id);`, table, table),
	}

	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec schema statement: %w", err)
		}
	}

	return nil
}
