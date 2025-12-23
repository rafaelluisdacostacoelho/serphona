# Analytics Processor Service — RAG Backlog (en-US)

## Status snapshot
- RAG indexer worker consumes `rag.ingestion.requested`, fetches HTTP/S3 with ETag, chunks, embeds, upserts pgvector.
- Circuit breaker + retries + DLQ in place; metrics counters/histograms exposed via Prometheus server (port `PROMETHEUS_PORT`, default 9000).
- Idempotency guard on `document_id+etag` via repo `etag_matches`.
- Unit tests cover retries, circuit open, embedding failure DLQ, and idempotent skip (`test_rag_worker_retries.py`).

## Open items (priority order)
1) **Pydantic v2 cleanup**: replace `.dict()` with `model_dump()`; silence deprecation warnings.
2) **Embedding cache**: optional Redis cache toggle for embeddings (hit/miss tests). Config flag + wiring into worker.
3) **Metrics wiring**: ensure `/metrics` is scraped (Helm values), add DLQ/lag alerts; consider phase latency buckets review.
4) **Idempotency tests vs real pgvector**: integration test (docker-compose.tests) validating `document_id+etag` skip and RLS/ACL schema alignment.
5) **Connectors**: REST/GraphQL/SAP ingest connectors with pagination/backoff and tenant filters; mark backlog when done.
6) **Content cleanup**: optional HTML/text normalization/language handling; enforce TTL/retention per namespace if configured.
7) **Runbooks**: DLQ drain/reprocess, reindex by document_id, handling embedding provider outages.

## Config to surface
- `PROMETHEUS_PORT` (metrics server), `MAX_RETRIES`, `RETRY_BACKOFF_SECONDS`, `DLQ_PATH`, `ETAG_CACHE_PATH`, `EMBEDDING_*`, `S3_*`, `ALLOWED_URI_SCHEMES`.

## Test coverage checklist
- [x] Retry exhaust → DLQ
- [x] Circuit open skip
- [x] Embedding failure → DLQ
- [x] ETag idempotent skip
- [ ] Integration: Kafka → worker → pgvector (docker-compose.tests)
- [ ] Redis embedding cache hit/miss

## Next suggested steps
- Tackle Pydantic v2 cleanup + embed cache toggle, then add integration test to close RAG indexing backlog items.
