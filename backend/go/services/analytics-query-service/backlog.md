# Analytics Query Service — Backlog (en-US)

## Status snapshot
 - Audit complete: service exposes unauthenticated Gin API, tenant_id is only a query param, ClickHouse queries are built with string params but no auth/RLS enforcement; rate limiter is IP/tenant header only; caching uses Redis with JSON but no invalidation on writes; no tracing/metrics/logging beyond prints; no billing/usage events; configs lack validation and TLS.

## Audit findings
 - Auth/RLS: no JWT or middleware; tenant comes from query; nothing prevents cross-tenant access; rate limiter not tied to authenticated tenant; cache keys include tenant but rely on caller honesty.
 - Input validation: time range defaults to last 7 days but no max window; query filters allow arbitrary limit/offset; topics array unparsed in SearchEvents; no pagination bounds; no namespace/channel filters.
 - Data layer: ClickHouse queries use positional params (safe) but no LIMIT caps besides defaults; SearchEvents count query ignores filters (agent/event type); no aggregation on partition keys; no retries/timeouts.
 - Cache: Redis cache logs to stdout; no TTL per data type; no invalidation on data refresh; cache key pattern uses tenant but no size limits.
 - Observability: no tracing, no Prometheus metrics (latency, rows, cache hits), no structured logging; health/ready minimal.
 - Security: CORS is default Gin; no TLS; no header sanitation; rate limiter per IP/tenant header without auth; no PII masking.
 - Resilience: no circuit breaker/timeouts for ClickHouse/Redis; server timeouts fixed; no backpressure; no graceful failure modes.
 - Billing/events: none for queries or rows scanned.
 - Docs/runbooks: README missing for en-US; no config matrix; no SLOs or dashboards.

## Action items
 1) Auth/RLS: require platform-auth middleware (JWT with tenant), enforce tenant filter on all queries/cache keys, and verify tenant in rate limiter; add namespace/channel filters if applicable.
 2) Config: centralize validated config (HTTP addr, ClickHouse DSN with TLS/auth, Redis cache optional, rate limits per tenant/IP, max window, default limit/offset caps), env prefix, and required checks.
 3) Validation/limits: enforce max time range, limit/offset bounds, allowed granularity values, and required tenant; validate topics list; clamp rate limits; add request payload schemas/OpenAPI.
 4) Observability: add tracing (HTTP + ClickHouse), Prometheus metrics (latency, rows, cache hits/misses, rate-limit blocks), structured logging with request/tenant IDs; health/ready with dependency checks.
 5) Cache: add size/TTL policy per endpoint, error handling without panics, optional bypass, and invalidation hooks; consider per-tenant quota.
 6) Resilience: add timeouts/retries/backoff for ClickHouse/Redis, optional circuit breaker, graceful degradation when cache/downstream unavailable.
 7) Billing/usage: emit events per query (rows scanned/returned, duration) tagged by tenant; integrate with billing/topic sink.
 8) Security: tighten CORS, add TLS support, sanitize headers, and consider PII masking in responses/logs.
 9) Docs/runbooks: add README/config matrix, SLOs, dashboards, and runbooks for high latency, cache flush, rate-limit tuning, ClickHouse issues.
 10) Testing: auth/tenant leakage tests, validation tests (time window/limits), cache hit/miss tests, metrics/tracing assertions, ClickHouse integration test (docker-compose) with seeded data, and load tests for p95/p99.

## Config to surface
 - HTTP_ADDR/TLS, CLICKHOUSE_HOST/PORT/DB/USER/PASSWORD/READ_TIMEOUT/WRITE_TIMEOUT/MAX_CONNS, REDIS_ADDR/PASSWORD/DB/CACHE_TTL/MAX_CACHE, RATE_LIMIT_PER_TENANT/IP, MAX_TIME_RANGE_DAYS, DEFAULT_LIMIT/MAX_LIMIT, METRICS_PORT/OTEL_EXPORTER, AUTH_ISSUER/AUDIENCE/SCOPES/TENANT_CLAIM, BILLING_TOPIC.

## Test coverage checklist
 - [ ] AuthZ + tenant filter enforcement (no cross-tenant)
 - [ ] Time range/limit validation and allowed granularity
 - [ ] Rate limiter per tenant/user/IP
 - [ ] ClickHouse queries with timeouts/retries and correct filters
 - [ ] Cache hit/miss + invalidation behavior
 - [ ] Metrics/tracing/billing events emitted

## Next suggested steps
 - Add auth middleware + validated config and tenant enforcement; then implement validation/limits and observability; harden cache and ClickHouse timeouts/retries; add billing events and tests/integration harness.
