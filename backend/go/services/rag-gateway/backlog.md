# RAG Gateway — Backlog (en-US)

## Status snapshot
- Audit complete: Gin API with stub tenant injection (no auth). Ingestion/query/namespace endpoints are basic; uses pgvector store and OpenAI embeddings when configured, otherwise stub errors. Validation is minimal; tenant defaults to header fallback. Event publishing is optional behind RAG_EVENTS_ENABLED with no retries/DLQ. No auth, rate limits, or request size caps. Config lacks validation; TLS and PGVector tuning absent. Observability uses platform-observability middleware if init succeeds but no metrics/logging in handlers. Namespaces are stubbed; migrations not wired.

## Audit findings
- Auth/tenant: no auth middleware; tenant pulled from header `X-Tenant-ID` or hardcoded "stub-tenant". No scope/namespace ACL checks; handlers trust payload values.
- Validation/limits: Gin binding only on some fields; handler duplicates tenant assignment; no max payload size, no allowed URI schemes, no tag/ACL length limits, no topK bounds. Namespace create/list stub returns NotImplemented.
- Ingestion: uses platform-rag metadata Normalize/Validate, embeds single chunk, upserts to pgvector. No chunking, no dedup/idempotency, no ETag/version conflict handling, no TTL enforcement beyond metadata. Errors return 501 Not Implemented.
- Query: embeds query and queries vector store; filters map passes through without validation; no score threshold; topK defaults but no max cap.
- Events: publishes `rag.ingestion.requested` only if RAG_EVENTS_ENABLED env true; no retry/backoff/DLQ; event includes metadata but lacks size/chunk counts; tracing IDs added when present.
- Config: env-driven defaults with no required checks (API key, pgvector dim/table, TopK bounds). No TLS or pool tuning beyond pgx defaults; PGVECTOR_LISTS unused. Embedding provider stub by default.
- Observability: optional OTEL middleware; handlers lack structured logs/metrics; health endpoints static.
- Security: no auth, no rate limiting, permissive error messages; no input size limits; openai client does not redact keys in errors; no HTTPS/mTLS options. Tenant fallback risks cross-tenant writes.
- Resilience: no retries around embeddings or vector store; no backpressure; no graceful shutdown hooks for publisher; events silently dropped when disabled.
- Testing: router tests only; usecase tests exist but coverage for failures, events, and tenant enforcement is minimal; no integration with Kafka/pgvector.

## Action items
1) Auth/tenant enforcement: integrate platform-auth middleware, require authenticated tenant and namespace ACL; remove stub tenant fallback; enforce tenant on store queries and events.
2) Input validation/limits: enforce max payload size, allowed URI schemes, tag/ACL length and counts, topK max, TTL bounds; ensure namespace is required and exists; reject empty content and oversize documents.
3) Ingestion pipeline: add chunking, dedup/idempotency (document_id + version/etag), conflict detection, and optional async queue. Persist namespace metadata and support TTL enforcement. Improve error codes (400/403/429/500 instead of 501).
4) Eventing: require RAG_EVENTS_ENABLED default true in prod; add retries/backoff and DLQ; include content length/chunk count; sign/trace events and add tenant/namespace labels. Handle publish failures with metrics and alerts.
5) Query path: add score threshold, filter validation/allowlist, pagination/offset for results, and max topK cap. Consider hybrid retrieval options if needed.
6) Config hardening: validate required envs (PGVECTOR_URL/table/dim, embedding API key when provider=openai), enforce sane ranges (topK, dim), add TLS/pool knobs for pgx, and expose request timeouts and rate limits.
7) Observability: add structured logging, Prometheus metrics (ingest/query latency, validations, embeddings errors, event publish success/fail), tracing spans in handlers, and correlation IDs. Add health/ready checks for db, embeddings, and publisher.
8) Security: tighten CORS if exposed, add rate limits per tenant/IP, request body size caps, and optional mTLS/TLS. Mask secrets in logs; sanitize error responses.
9) Namespaces: implement namespace list/create backed by store or tenant-manager; enforce ownership and quotas.
10) Testing/runbooks: expand unit tests for validation/error paths, event flag behavior, embedding failures, and topK bounds; integration test with pgvector and Kafka fake; runbook for embedding outages and event pipeline failures.

## Config to surface
- HTTP addr, timeouts, body size, CORS/rate limits; auth issuer/audience/scopes and tenant claim.
- PGVECTOR URL/table/dim/lists, TLS/pooling; embedding provider/model/base URL/api key/timeouts.
- RAG_EVENTS_ENABLED, Kafka publisher config, retries/backoff/DLQ topic.
- Validation limits: max payload size, topK max/default, allowed URI schemes, tag/ACL limits, TTL bounds.
- Observability exporters (OTEL/Prometheus), log level/format.

## Test coverage checklist
- [ ] Auth/tenant enforcement and namespace ACL
- [ ] Validation limits (payload size, URI schemes, topK caps, TTL/tag/ACL bounds)
- [ ] Ingestion idempotency/chunking and conflict handling
- [ ] Event publish flag/retry/DLQ behavior
- [ ] Metrics/tracing/logging emitted; pgvector and embedding error handling

## Next suggested steps
- Add auth/tenant middleware and validated config with limits; wire namespace storage; implement idempotent ingestion with pgvector and reliable event publishing (retries/DLQ), plus metrics/tracing and rate limits.
