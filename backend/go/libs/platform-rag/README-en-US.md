# platform-rag

Shared RAG building blocks for multi-tenant services (ingestion, indexing, retrieval).

## What’s here
- model/: typed Chunk, Query, Filter with validation (tenant/ns required, TTL bounds, TopK limits, embedding dims >0).
- metadata/: normalize/validate metadata (dedupe tags/ACL, clamp TTL 0..1y, URI sanity, canonical keys).
- chunking/: boundary-aware text splitting with overlap; options for max chars, overlap, language hints, max segments, whitespace normalization.
- embedding/: adapters (OpenAI) with token/dimension guards and retries; retry/cache decorators; interfaces for new providers.
- vector/: Store interface; pgvector store with schema helper and Prometheus observer; reranker decorator to reorder results.

## Usage overview
- Chunk and validate: use chunking.DefaultOptions(); validate chunks and metadata before persisting.
- Embeddings: pick an embedding client, optionally wrap with RetryableClient and CachingClient.
- Store: configure pgvector (URL, TableName, Dimension, Lists, Observer); run CreateSchema once; use NewStore for UpsertChunks/Query.
- Filters: support tags/acl/source/version/channel/language plus MinScore/TopK to control retrieval.
- Rerank: wrap store with reranker.Store to apply external reranker while preserving vector scores.

## Quick starts
- Integration test (pgvector): set TEST_PGVECTOR_DSN; run `go test ./vector/pgvector -tags=integration` or `make rag-pgvector-int`.
- Chunking helper: `segments := chunking.ChunkText(text, chunking.DefaultOptions())`.
- Prometheus metrics: provide observer via pgvector.Config{Observer: pgvector.NewPrometheusObserver()}.

## Testing
- Unit: go test ./...
- Integration: go test ./vector/pgvector -tags=integration (requires Postgres+pgvector).

See IMPLEMENTATION_GUIDE-en-US.md for deeper integration patterns.
