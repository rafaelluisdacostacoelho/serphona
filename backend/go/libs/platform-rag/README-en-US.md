# platform-rag

Shared RAG building blocks for multi-tenant services (ingestion/index/retrieval).

What’s included (usable):
- Typed models for chunks/queries with validation and sane defaults (`pkg/model`).
- Metadata normalization/validation (`pkg/metadata`) — dedupe tags/ACL, mirror canonical fields, guard TTL (0..1y) and basic URI sanity.
- Text chunking helper with overlap and boundary-aware splits (`pkg/chunking`).
- Vector store interface plus pgvector implementation with schema helper (`pkg/vector`).
- Embedding adapters: OpenAI client with token guards, retries/backoff, dimension checks; retry/caching decorators in `pkg/embedding`.
- Reranker decorator: wrap a `vector.Store` with `pkg/vector/reranker.Store` to apply custom reranker ordering.
- Filters support language/channel fields for richer querying.

Planned/next:
- Observability hooks (metrics/traces) around store ops.
- Additional embedding client adapters and filter/rerank decorators.

## Quickstart

```go
cfg := pgvector.Config{
	URL:       os.Getenv("PG_DSN"),
	TableName: "rag_chunks",
	Dimension: 1536,
	Lists:     200,
}

db, _ := sql.Open("postgres", cfg.URL)
_ = pgvector.CreateSchema(context.Background(), db, cfg)

store := pgvector.NewStore(db, cfg)

chunk := model.Chunk{
	TenantID:   "tenant-1",
	Namespace:  "default",
	DocumentID: "doc-1",
	ChunkID:    "ch-1",
	Content:    "The quick brown fox",
	Embedding:  embeddingVector,
	Metadata: model.ChunkMetadata{Source: "kb", Tags: []string{"faq"}},
}

_ = store.UpsertChunks(ctx, []model.Chunk{chunk})

q := model.Query{TenantID: "tenant-1", Namespace: "default", QueryVector: queryEmbedding, TopK: 5, MinScore: 0.2}
results, _ := store.Query(ctx, q)
```

### Integration test (optional)
- Requires Postgres with pgvector extension.
- Set `TEST_PGVECTOR_DSN` and run: `go test ./vector/pgvector -tags=integration`.
- Or use repo root helper: `make rag-pgvector-int` (starts pgvector via docker-compose.tests.yml, runs the integration suite, then stops the container).

### Metrics (Prometheus)
- Use `pgvector.NewPrometheusObserver(...)` and pass it via `Config{Observer: obs}` to emit `platform_rag_pgvector_requests_total` and `platform_rag_pgvector_latency_seconds` with labels `operation` and `status` (ok|error).

### Embeddings
- OpenAI adapter: construct with `NewOpenAIAdapter(client, tokenizer, OpenAIConfig{Model: ..., MaxTokens: ..., ExpectedDim: ...})`; it retries transient errors and enforces token/dimension checks.
- Decorators: `RetryableClient` for generic retries/backoff and `CachingClient` with in-memory TTL cache via `NewInMemoryCache()`.
```

## Schema contract (pgvector)
- Table: default `rag_chunks` (configurable).
- Columns: tenant_id, namespace, document_id, chunk_id (PK), content, metadata (jsonb), embedding (vector), etag, created_at.
- Indexes: PK on (tenant_id, namespace, chunk_id), ivfflat index on embedding (lists configurable), lookup on (tenant_id, namespace, document_id).
- RLS: enforce per-tenant policies in Postgres for production usage.

## Chunking helper

```go
segments := chunking.ChunkText(text, chunking.Options{MaxChars: 1200, Overlap: 80, NormalizeWhitespace: true, TrimSpace: true})
```

Defaults target ingestion worker expectations (1200 chars, 80 overlap, boundary-aware). Use `chunking.DefaultOptions()` for a pre-set configuration.
`LanguageCode` can hint sentence delimiters for some languages (e.g., zh/ja use 。！？), and `MaxSegments` caps chunk count for very long inputs.
Set `MaxSegments` to cap the number of chunks for extremely long inputs.
