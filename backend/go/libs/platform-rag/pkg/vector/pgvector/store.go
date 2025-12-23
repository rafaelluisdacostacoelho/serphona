package pgvector

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/pkg/model"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/pkg/vector"
)

// Config holds connection and schema settings for pgvector.
type Config struct {
	URL       string
	TableName string
	Dimension int
	Lists     int
}

// Store is a stub implementation backed by pgvector.
type Store struct {
	db     *sql.DB
	config Config
}

// NewStore builds a Store with the given DB handle and config.
func NewStore(db *sql.DB, config Config) Store {
	return Store{db: db, config: config}
}

// Ensure Store satisfies vector.Store.
var _ vector.Store = (*Store)(nil)

// UpsertChunks persists chunks into pgvector using an upsert on (tenant_id, namespace, chunk_id).
func (s Store) UpsertChunks(ctx context.Context, chunks []model.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	table, err := s.tableName()
	if err != nil {
		return err
	}

	dim := s.dimension()
	if dim <= 0 {
		return errors.New("pgvector dimension must be > 0")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query := fmt.Sprintf(`
INSERT INTO %s (tenant_id, namespace, document_id, chunk_id, content, metadata, embedding, etag, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (tenant_id, namespace, chunk_id)
DO UPDATE SET content=EXCLUDED.content, metadata=EXCLUDED.metadata, embedding=EXCLUDED.embedding, etag=EXCLUDED.etag, created_at=EXCLUDED.created_at;
`, table)

	for _, ch := range chunks {
		if ch.TenantID == "" || ch.Namespace == "" || ch.ChunkID == "" {
			return errors.New("tenant_id, namespace, and chunk_id are required")
		}
		if len(ch.Embedding) != dim {
			return fmt.Errorf("embedding dimension mismatch: got %d, expected %d", len(ch.Embedding), dim)
		}

		meta, err := json.Marshal(ch.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}

		created := ch.CreatedAt
		if created.IsZero() {
			created = time.Now().UTC()
		}

		vec := pgvector.NewVector(ch.Embedding)

		if _, err := tx.ExecContext(ctx, query,
			ch.TenantID,
			ch.Namespace,
			ch.DocumentID,
			ch.ChunkID,
			ch.Content,
			meta,
			vec,
			ch.ETag,
			created,
		); err != nil {
			return fmt.Errorf("upsert chunk %s: %w", ch.ChunkID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// Query searches chunks via pgvector using distance ordering.
func (s Store) Query(ctx context.Context, q model.Query) ([]model.Chunk, error) {
	if q.TenantID == "" || q.Namespace == "" {
		return nil, errors.New("tenant_id and namespace are required")
	}

	if len(q.QueryVector) == 0 {
		return nil, errors.New("query_vector is required")
	}

	dim := s.dimension()
	if len(q.QueryVector) != dim {
		return nil, fmt.Errorf("query_vector dimension mismatch: got %d, expected %d", len(q.QueryVector), dim)
	}

	table, err := s.tableName()
	if err != nil {
		return nil, err
	}

	limit := q.TopK
	if limit <= 0 {
		limit = 5
	}

	filtersJSON, err := json.Marshal(q.Filters)
	if err != nil {
		return nil, fmt.Errorf("marshal filters: %w", err)
	}
	if len(filtersJSON) == 2 { // "{}"
		filtersJSON = nil
	}

	vec := pgvector.NewVector(q.QueryVector)

	query := fmt.Sprintf(`
SELECT tenant_id, namespace, document_id, chunk_id, content, metadata, etag, created_at, (embedding <-> $3) AS distance
FROM %s
WHERE tenant_id = $1 AND namespace = $2
AND ($4::jsonb IS NULL OR metadata @> $4::jsonb)
ORDER BY embedding <-> $3
LIMIT $5;
`, table)

	rows, err := s.db.QueryContext(ctx, query, q.TenantID, q.Namespace, vec, filtersJSON, limit)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var result []model.Chunk
	for rows.Next() {
		var ch model.Chunk
		var metaBytes []byte
		var distance float64
		if err := rows.Scan(&ch.TenantID, &ch.Namespace, &ch.DocumentID, &ch.ChunkID, &ch.Content, &metaBytes, &ch.ETag, &ch.CreatedAt, &distance); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if len(metaBytes) > 0 {
			_ = json.Unmarshal(metaBytes, &ch.Metadata)
		}
		// Convert distance to a simple score heuristic.
		ch.Score = 1 / (1 + distance)
		result = append(result, ch)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}

	return result, nil
}

// Ping checks connectivity.
func (s Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s Store) tableName() (string, error) {
	name := s.config.TableName
	if name == "" {
		name = "rag_chunks"
	}
	if !tablePattern.MatchString(name) {
		return "", fmt.Errorf("invalid table name: %s", name)
	}
	return name, nil
}

func (s Store) dimension() int {
	if s.config.Dimension > 0 {
		return s.config.Dimension
	}
	return 1536
}

var tablePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
