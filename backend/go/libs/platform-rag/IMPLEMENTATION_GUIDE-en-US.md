# IMPLEMENTATION_GUIDE (platform-rag)

## Purpose
How to embed platform-rag components (chunking, metadata validation, embeddings, vector stores) into multi-tenant services while keeping schemas, observability, and tenancy intact.

## Core components
- **Models** (`model`): typed `Chunk`, `Query`, `Filter`, with validation (tenant/namespace required, TTL bounds, min TopK, embedding dims >0).
- **Metadata** (`metadata`): normalize/validate chunk metadata (dedupe tags/ACL, clamp TTL 0..1y, basic URI checks, canonical keys).
- **Chunking** (`chunking`): boundary-aware text splitting with overlap; `Options{MaxChars, Overlap, LanguageCode, MaxSegments, NormalizeWhitespace, TrimSpace}`; defaults align to ingestion worker (1200 chars, 80 overlap).
- **Embeddings** (`embedding`): OpenAI adapter with token/dimension guards and retries; retry and in-memory cache decorators; interfaces to add new providers.
- **Vector store** (`vector`): `Store` interface (UpsertChunks, Query); pgvector implementation + `pgvector.CreateSchema` helper and Prometheus observer; reranker decorator to reorder results via external reranker.

## Integration steps (server flow)
1) **Chunk & validate input**: use `chunking.ChunkText` or `ChunkDocument` with `chunking.DefaultOptions()`; validate chunks via `model.ValidateChunk` and metadata via `metadata.Normalize`/`Validate`.
2) **Embed**: choose an embedding client (e.g., `embedding.OpenAIAdapter`), optionally wrap with `RetryableClient` and `CachingClient` for resilience and cost control.
3) **Store**: configure pgvector `Config{URL, TableName, Dimension, Lists, Observer}`; run `pgvector.CreateSchema(ctx, db, cfg)` once; then `store := pgvector.NewStore(db, cfg)`.
4) **Upsert/query**: call `store.UpsertChunks(ctx, chunks)` and `store.Query(ctx, query)`; include `Filter` (tags/acl/source/version/channel/language) and `MinScore/TopK` to shape results.
5) **Rerank (optional)**: wrap the store with `vector/reranker.Store` to re-order results using an external reranker while preserving underlying vector scores.
6) **Observability**: enable Prometheus observer in pgvector config to emit request/latency metrics labeled by operation/status.

## Tenancy and safety
- Always set `TenantID` and `Namespace` on chunks and queries; enforce RLS on Postgres in production.
- Table naming: use unique tables per environment/namespace if needed; `CreateSchema` validates names.
- Chunk TTL: bounded to one year; zero disables expiry. Apply TTL in your retention jobs or DB policies.

## Error handling
- Embedding adapters return typed errors for token/dimension violations and propagate provider errors.
- Vector store operations surface errors directly; wrap with retries at caller if needed.
- Reranker decorator returns underlying store errors unchanged.

## Testing
- Unit tests: `go test ./...` (pgx/DB interactions mocked where needed).
- Integration (pgvector): set `TEST_PGVECTOR_DSN` and run `go test ./vector/pgvector -tags=integration` or `make rag-pgvector-int` (uses docker-compose.tests.yml).
- Chunking/metadata tests cover boundary conditions (overlap, MaxSegments, TTL bounds, URI sanity).

## Extension points
- Implement new embedding providers by satisfying `embedding.Client`.
- Add observability by supplying your own `vector.Observer` implementation.
- Swap reranker strategy by providing a new reranker to `reranker.Store`.

## Example wiring
```go
cfg := pgvector.Config{URL: dsn, TableName: "rag_chunks", Dimension: 1536, Lists: 200, Observer: pgvector.NewPrometheusObserver()}
_ = pgvector.CreateSchema(ctx, db, cfg)
store := pgvector.NewStore(db, cfg)
emb := embedding.NewOpenAIAdapter(client, tokenizer, embedding.OpenAIConfig{Model: "text-embedding-3-large", MaxTokens: 8192, ExpectedDim: 1536})
chunks := chunking.ChunkText(text, chunking.DefaultOptions())
_ = store.UpsertChunks(ctx, chunks)
res, _ := store.Query(ctx, model.Query{TenantID: "t1", Namespace: "default", QueryVector: vec, TopK: 5, MinScore: 0.2})
```
