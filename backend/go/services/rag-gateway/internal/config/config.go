package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds service configuration.
type Config struct {
	HTTPAddr          string
	PGVectorURL       string
	PGVectorTable     string
	PGVectorDim       int
	PGVectorLists     int
	TopKDefault       int
	EmbeddingProvider string
	EmbeddingModel    string
	EmbeddingAPIKey   string
	EmbeddingBaseURL  string
}

// Load reads configuration from environment with defaults.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:          getEnv("HTTP_ADDR", ":8086"),
		PGVectorURL:       getEnv("PGVECTOR_URL", "postgresql://postgres:postgres@localhost:5432/serphona_rag?sslmode=disable"),
		PGVectorTable:     getEnv("PGVECTOR_TABLE", "rag_chunks"),
		PGVectorDim:       1536,
		PGVectorLists:     100,
		TopKDefault:       5,
		EmbeddingProvider: getEnv("EMBEDDING_PROVIDER", "noop"),
		EmbeddingModel:    getEnv("EMBEDDING_MODEL", "text-embedding-3-small"),
		EmbeddingAPIKey:   os.Getenv("EMBEDDING_API_KEY"),
		EmbeddingBaseURL:  os.Getenv("EMBEDDING_BASE_URL"),
	}

	if dimStr := os.Getenv("PGVECTOR_DIM"); dimStr != "" {
		dim, err := strconv.Atoi(dimStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid PGVECTOR_DIM: %w", err)
		}
		cfg.PGVectorDim = dim
	}

	if topkStr := os.Getenv("RAG_TOPK_DEFAULT"); topkStr != "" {
		tk, err := strconv.Atoi(topkStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid RAG_TOPK_DEFAULT: %w", err)
		}
		cfg.TopKDefault = tk
	}

	if listsStr := os.Getenv("PGVECTOR_LISTS"); listsStr != "" {
		lists, err := strconv.Atoi(listsStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid PGVECTOR_LISTS: %w", err)
		}
		cfg.PGVectorLists = lists
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
