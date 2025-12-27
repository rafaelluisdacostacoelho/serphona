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

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/model"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/vector"
)

// Config holds connection and schema settings for pgvector.
type Config struct {
	URL       string
	TableName string
	Dimension int
	Lists     int
	Timeout   time.Duration
	Observer  Observer
}

// Store is a stub implementation backed by pgvector.
type Store struct {
	db       *sql.DB
	config   Config
	observer Observer
}

// NewStore builds a Store with the given DB handle and config.
func NewStore(db *sql.DB, config Config) Store {
	obs := config.Observer
	if obs == nil {
		obs = NoopObserver{}
	}
	return Store{db: db, config: config, observer: obs}
}

// Ensure Store satisfies vector.Store.
var _ vector.Store = (*Store)(nil)

// UpsertChunks persists chunks into pgvector using an upsert on (tenant_id, namespace, chunk_id).
func (s Store) UpsertChunks(ctx context.Context, chunks []model.Chunk) (err error) {
	ctx, cancel := withTimeout(ctx, s.config.Timeout)
	defer cancel()
	ctx, endTrace := s.observer.Trace(ctx, "pgvector.upsert")
	start := time.Now()
	defer func() {
		s.observer.RecordLatency(ctx, "pgvector.upsert", time.Since(start), err)
		endTrace(err)
	}()

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

	for i := range chunks {
		chunks[i].Metadata.Normalize()
		if err := model.ValidateChunk(chunks[i], dim); err != nil {
			return err
		}
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
		metaMap := encodeMetadata(ch.Metadata)
		meta, _ := json.Marshal(metaMap)

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
		return StoreError{Op: "upsert", Err: fmt.Errorf("commit: %w", err)}
	}

	return nil
}

// Query searches chunks via pgvector using distance ordering.
func (s Store) Query(ctx context.Context, q model.Query) (result []model.Chunk, err error) {
	ctx, cancel := withTimeout(ctx, s.config.Timeout)
	defer cancel()
	ctx, endTrace := s.observer.Trace(ctx, "pgvector.query")
	start := time.Now()
	defer func() {
		s.observer.RecordLatency(ctx, "pgvector.query", time.Since(start), err)
		endTrace(err)
	}()

	q.Normalize()
	dim := s.dimension()
	if err := q.Validate(dim); err != nil {
		return nil, err
	}

	table, err := s.tableName()
	if err != nil {
		return nil, err
	}

	limit := q.TopK

	filtersJSON, _ := marshalFilters(q.Filters)

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
		return nil, StoreError{Op: "query", Err: fmt.Errorf("query: %w", err)}
	}
	defer rows.Close()

	for rows.Next() {
		var ch model.Chunk
		var metaBytes []byte
		var distance float64
		if err := rows.Scan(&ch.TenantID, &ch.Namespace, &ch.DocumentID, &ch.ChunkID, &ch.Content, &metaBytes, &ch.ETag, &ch.CreatedAt, &distance); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if len(metaBytes) > 0 {
			_ = json.Unmarshal(metaBytes, &ch.Metadata.Attributes)
			ch.Metadata.Normalize()
		}
		// Convert distance to a simple score heuristic.
		score := 1 / (1 + distance)
		if q.MinScore > 0 && score < q.MinScore {
			continue
		}
		ch.Score = score
		result = append(result, ch)
	}

	if err := rows.Err(); err != nil {
		return nil, StoreError{Op: "query", Err: fmt.Errorf("rows err: %w", err)}
	}

	return result, nil
}

// Ping checks connectivity.
func (s Store) Ping(ctx context.Context) (err error) {
	ctx, cancel := withTimeout(ctx, s.config.Timeout)
	defer cancel()
	ctx, endTrace := s.observer.Trace(ctx, "pgvector.ping")
	start := time.Now()
	defer func() {
		s.observer.RecordLatency(ctx, "pgvector.ping", time.Since(start), err)
		endTrace(err)
	}()

	err = s.db.PingContext(ctx)
	if err != nil {
		return StoreError{Op: "ping", Err: err}
	}
	return nil
}

// StoreError wraps store operations with an Op label for observability correlation.
type StoreError struct {
	Op  string
	Err error
}

func (e StoreError) Error() string {
	return fmt.Sprintf("pgvector %s: %v", e.Op, e.Err)
}

func (e StoreError) Unwrap() error { return e.Err }

func encodeMetadata(meta model.ChunkMetadata) map[string]any {
	meta.Normalize()
	data := map[string]any{}
	for k, v := range meta.Attributes {
		data[k] = v
	}
	if len(meta.Tags) > 0 {
		data["tags"] = meta.Tags
	}
	if len(meta.ACL) > 0 {
		data["acl"] = meta.ACL
	}
	if meta.Version != "" {
		data["version"] = meta.Version
	}
	if meta.Source != "" {
		data["source"] = meta.Source
	}
	if meta.URI != "" {
		data["uri"] = meta.URI
	}
	if meta.TTLSeconds > 0 {
		data["ttl_seconds"] = meta.TTLSeconds
	}
	return data
}

func marshalFilters(f model.Filters) ([]byte, error) {
	f.Normalize()
	data := map[string]any{}
	for k, v := range f.Attributes {
		data[k] = v
	}
	if len(f.Tags) > 0 {
		data["tags"] = f.Tags
	}
	if len(f.ACL) > 0 {
		data["acl"] = f.ACL
	}
	if f.Source != "" {
		data["source"] = f.Source
	}
	if f.Version != "" {
		data["version"] = f.Version
	}
	if f.URI != "" {
		data["uri"] = f.URI
	}
	if f.Language != "" {
		data["language"] = f.Language
	}
	if f.Channel != "" {
		data["channel"] = f.Channel
	}

	if len(data) == 0 {
		return nil, nil
	}

	buf, _ := json.Marshal(data)
	return buf, nil
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
	if s.config.Dimension < 0 {
		return s.config.Dimension
	}
	if s.config.Dimension == 0 {
		return 1536
	}
	return s.config.Dimension
}

var tablePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
