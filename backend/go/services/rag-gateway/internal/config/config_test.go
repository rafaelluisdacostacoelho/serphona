package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	os.Clearenv()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.HTTPAddr != ":8086" {
		t.Fatalf("expected default HTTP_ADDR :8086, got %s", cfg.HTTPAddr)
	}
	if cfg.PGVectorTable != "rag_chunks" {
		t.Fatalf("expected default table rag_chunks, got %s", cfg.PGVectorTable)
	}
	if cfg.PGVectorDim != 1536 {
		t.Fatalf("expected default dim 1536, got %d", cfg.PGVectorDim)
	}
	if cfg.PGVectorLists != 100 {
		t.Fatalf("expected default lists 100, got %d", cfg.PGVectorLists)
	}
	if cfg.TopKDefault != 5 {
		t.Fatalf("expected default topk 5, got %d", cfg.TopKDefault)
	}
	if cfg.EmbeddingProvider != "noop" {
		t.Fatalf("expected default embedding provider noop, got %s", cfg.EmbeddingProvider)
	}
	if cfg.EmbeddingModel != "text-embedding-3-small" {
		t.Fatalf("expected default embedding model text-embedding-3-small, got %s", cfg.EmbeddingModel)
	}
	if cfg.EmbeddingAPIKey != "" {
		t.Fatalf("expected empty embedding api key by default")
	}
}

func TestLoadOverrides(t *testing.T) {
	os.Clearenv()
	t.Setenv("HTTP_ADDR", ":9999")
	t.Setenv("PGVECTOR_DIM", "128")
	t.Setenv("PGVECTOR_LISTS", "250")
	t.Setenv("RAG_TOPK_DEFAULT", "9")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.HTTPAddr != ":9999" {
		t.Fatalf("expected HTTP_ADDR override :9999, got %s", cfg.HTTPAddr)
	}
	if cfg.PGVectorDim != 128 {
		t.Fatalf("expected dim 128, got %d", cfg.PGVectorDim)
	}
	if cfg.PGVectorLists != 250 {
		t.Fatalf("expected lists 250, got %d", cfg.PGVectorLists)
	}
	if cfg.TopKDefault != 9 {
		t.Fatalf("expected topk 9, got %d", cfg.TopKDefault)
	}
}
