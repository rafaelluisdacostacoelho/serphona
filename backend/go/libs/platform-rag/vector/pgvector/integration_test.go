//go:build integration

package pgvector

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/model"
)

func TestIntegrationUpsertQuery(t *testing.T) {
	dsn := os.Getenv("TEST_PGVECTOR_DSN")
	if dsn == "" {
		t.Skip("TEST_PGVECTOR_DSN not set; skipping integration test")
	}

	table := "rag_chunks_int_test"
	cfg := Config{URL: dsn, TableName: table, Dimension: 3, Lists: 10, Timeout: 5 * time.Second}

	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	_ = dropTable(ctx, db, table)

	if err := CreateSchema(ctx, db, cfg); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	store := NewStore(db, cfg)

	chunks := []model.Chunk{{
		TenantID:   "t1",
		Namespace:  "ns1",
		DocumentID: "doc1",
		ChunkID:    "c1",
		Content:    "hello world",
		Embedding:  []float32{0.1, 0.2, 0.3},
		Metadata:   model.ChunkMetadata{Tags: []string{"faq"}, Attributes: map[string]string{"lang": "en"}},
	}}

	if err := store.UpsertChunks(ctx, chunks); err != nil {
		t.Fatalf("upsert chunks: %v", err)
	}

	q := model.Query{TenantID: "t1", Namespace: "ns1", QueryVector: []float32{0.1, 0.2, 0.3}, TopK: 5, MinScore: 0.0}
	res, err := store.Query(ctx, q)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	if len(res[0].Metadata.Tags) == 0 || res[0].Metadata.Tags[0] != "faq" {
		t.Fatalf("tags not round-tripped: %+v", res[0].Metadata)
	}

	q.Filters = model.Filters{Attributes: map[string]string{"lang": "en"}, Tags: []string{"faq"}}
	res, err = store.Query(ctx, q)
	if err != nil {
		t.Fatalf("query with filters: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result when filters match, got %d", len(res))
	}

	q.Filters = model.Filters{Attributes: map[string]string{"lang": "es"}}
	res, err = store.Query(ctx, q)
	if err != nil {
		t.Fatalf("query with mismatched filters: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("expected 0 results when filters do not match, got %d", len(res))
	}

	// min_score should filter out low scores if set high.
	q.Filters = model.Filters{}
	q.MinScore = 0.99
	res, err = store.Query(ctx, q)
	if err != nil {
		t.Fatalf("query with min_score: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("expected 0 results with high min_score, got %d", len(res))
	}

	// Dimension mismatch should error.
	bad := []model.Chunk{{
		TenantID:  "t1",
		Namespace: "ns1",
		ChunkID:   "bad",
		Content:   "oops",
		Embedding: []float32{1, 2},
	}}
	if err := store.UpsertChunks(ctx, bad); err == nil {
		t.Fatalf("expected dimension mismatch error")
	}

	if err := dropTable(ctx, db, table); err != nil {
		t.Fatalf("cleanup drop table: %v", err)
	}
}

func dropTable(ctx context.Context, db *sql.DB, table string) error {
	_, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS "+table)
	return err
}
