# rag-processor-service — Backlog (en-US)

## Status snapshot
- Pure scaffold: only `main.py` logs a stub message; no config, Kafka consumer/producer, storage, embedding, or metrics. Tests contain only a placeholder import check.
- Requirements are empty; no runtime deps declared; no Docker/k8s manifests; no CI wiring.

## Critical gaps to fill before shipping
- **Config & bootstrap**: add Pydantic/BaseSettings config (Kafka, ClickHouse/pgvector, embedding provider, tracing/metrics, HTTP health). Provide `.env.example`.
- **Ingestion pipeline**: Kafka consumer for RAG events with backpressure, retries, DLQ; idempotency by `document_id+etag`/content hash; per-tenant isolation.
- **Embedding & storage**: pluggable embedding providers with timeouts/retries; write vectors/metadata to pgvector or ClickHouse; chunking/token limits; enforce tenant_id and namespace.
- **Observability**: Prometheus metrics (ingested, processed, failures, latency, lag), structured logging, tracing spans; health/readiness endpoints with dependency checks.
- **Security & governance**: input validation, size limits, allowlist of URI schemes, secrets via env/secret manager, TLS/SASL for Kafka/DB, optional RLS per tenant.
- **Operations**: DLQ replay/runbook, reindex by document_id, schema migration strategy, retention/TTL on storage, sampling controls for expensive embeddings.
- **Testing**: pytest with fakes for Kafka and DB; contract test for idempotency; load test profile for high-volume ingestion; lint/CI wiring.

## Configuration to surface
- Kafka bootstrap/topics/group, TLS/SASL, backoff/retries, max batch size, DLQ topic.
- Storage (pgvector/ClickHouse) endpoints/creds, table/index names, TTL/retention, batch size.
- Embedding provider/model/API key, timeout, concurrency, cache toggle.
- Metrics/tracing ports, log level, health readiness checks.

## Test coverage targets
- Deserialization/validation failures route to DLQ without committing offsets.
- Idempotent skip on identical `document_id+etag` or content hash; chunking/tokenization correctness.
- Embedding provider timeout/retry/backoff; DLQ after exhaust.
- Storage write errors surfaced with metrics and no data loss/duplication.
- Readiness fails when Kafka/DB unreachable; metrics exposed.

## Next steps
- Introduce config + dependency wiring, then implement consumer → embed → storage pipeline with DLQ/idempotency and metrics. Add pytest fakes and hook into CI.
