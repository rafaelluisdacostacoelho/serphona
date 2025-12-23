# Analytics Processor Service — Backlog (en-US)

## Status snapshot (code now)
- Kafka consumer (`voc-events` by default) ingests JSON into ClickHouse via `ConsumerWorker`; batches 500 or 2s. No DLQ or error metrics; bad messages are silently dropped after JSON/Pydantic parse failure.
- NLP step is a rule-based sentiment heuristic with in-process cache only; no model calls, language detection, or topic extraction. Sentiment/topic optional fields are often `None`.
- ClickHouse table `voc_events` (MergeTree) is created at startup; partitioned by `(tenant_id, month)` with no TTL, no RLS/ACL enforcement, and free-form payload persisted as JSON string.
- FastAPI app exposes `/healthz` and `/metrics/summary`; no auth/tenant scoping and the summary query concatenates `tenant_id` directly into SQL (injection risk).
- Settings limited to Kafka/ClickHouse/HTTP port; no TLS/SASL for Kafka, no ClickHouse TLS, no observability/tracing, no auth/quotas/rate limits.

## Critical gaps to address
1) **Schema & validation**: Enforce structured event schemas per type (call/agent_action/tool/qos). Make `tenant_id` UUID, validate timestamps, and reject/route to DLQ instead of silent drop; log with metrics.
2) **DLQ & retries**: Add DLQ (Kafka topic or object store) for deserialization/processing failures; metric errors and add backoff around NLP/repo writes. Ensure stop/flush commits offsets deterministically.
3) **NLP pipeline**: Replace heuristic with configurable model/provider; add language detection, topic/keyphrase extraction, PII masking; move cache to Redis with TTL and tenant-aware keys.
4) **ClickHouse hygiene**: Add TTL/retention, compression, and type constraints; consider RLS or tenant-specific databases; validate `payload` size; handle clock skew; parameterize table name/prefix.
5) **Security & auth**: Protect `/metrics/summary` and sanitize `tenant_id` (parameterized query). Add service auth/TLS for Kafka/ClickHouse, and rate limiting on HTTP endpoints.
6) **Observability**: Expose processing/error metrics, consumer lag, DLQ counts; add tracing spans for consume → NLP → ClickHouse; structured logs with tenant/call IDs.
7) **Testing**: Current tests target old RAG path; add unit/contract tests for ConsumerWorker (decode error → DLQ, batch commit), NLP pipeline outputs, ClickHouse repo schema/TTL, and API SQL injection guard.
8) **Operations/runbooks**: Document DLQ replay, lag investigation, ClickHouse schema migrations/TTL changes, and cache warm-up strategy.

## Configuration to surface/guard
- Kafka: bootstrap, group, topics, TLS/SASL, consumer timeouts, max in-flight, backoff, DLQ topic.
- ClickHouse: TLS, credentials, table/TTL/retention, write batch size, retries, compression.
- NLP: provider/model, cache backend/TTL, language, limits on transcript length.
- HTTP: auth mode, rate limits, allowed origins; metrics/health ports.

## Test coverage targets
- Deserialization failures → DLQ (no commit) and success path commits highest offsets.
- Batch processing with retry/backoff + ClickHouse insert errors.
- NLP outputs per event type and cache hit/miss with Redis fake.
- SQL injection protection and tenant scoping on `/metrics/summary`.
- TTL/retention honored in ClickHouse integration test (docker-compose.tests).

## Next steps
- Implement DLQ + error metrics; parameterize/secure summary endpoint; add structured validation and tenant UUID enforcement.
- Swap heuristic NLP for configurable provider + Redis cache; add ClickHouse TTL and optional RLS strategy.
- Refresh test suite to cover current worker/NLP/ClickHouse/API flows and align CI docker-compose.tests profile.
