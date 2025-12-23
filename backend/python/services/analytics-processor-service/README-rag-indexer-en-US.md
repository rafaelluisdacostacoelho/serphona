# RAG Indexer (stub)

Skeleton worker to consume `rag.ingestion.requested`, fetch content, chunk, and upsert into pgvector.

Status: scaffold with HTTPS/S3 fetcher and pg upsert including embeddings (noop/OpenAI configurable).

## How it works (current)
- Consume Kafka topic `rag.ingestion.requested` (feature-flagged publisher in rag-gateway).
- Validate payload via Pydantic (`RAGIngestionRequested`).
- Fetch blob from `uri` (HTTPS, S3/MinIO) with allowed schemes; dedupe via `etag`/If-None-Match.
- Chunk text (~1.2k chars, 80 overlap, natural boundaries) and upsert pg table with embeddings (vector column) and tenant/namespace filters.
- Commit offsets after successful batch; DLQ/retry not implemented.

## Next steps
- Add retry/backoff + DLQ; cache `etag` between runs.
- Enforce RLS/tenant policies and validation on pg connection.
- Add metrics (Prometheus) and tracing hooks.
- Add unit tests mocking consumer/repo and chunking pipeline.
