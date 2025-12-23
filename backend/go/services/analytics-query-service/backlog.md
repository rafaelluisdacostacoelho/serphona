# Analytics Query Service — Backlog (en-US)

## Status snapshot
- Provides query APIs over analytics data (likely ClickHouse/PG). Current implementation not audited here; update once reviewed.
- Must enforce tenant_id/RLS on all queries and expose filters (namespace/time range) safely.

## Open items
1) **Auth & RLS**: require JWT with tenant_id; enforce tenant filters at query layer; add tests to prevent cross-tenant leakage.
2) **Query contracts**: validate inputs (date ranges, namespace, pagination); limit result size; add schema/contract tests.
3) **Performance**: tune queries and indexes; consider query cache (Redis) with invalidation by document_id/namespace.
4) **Observability**: tracing around queries; metrics for latency, rows returned, cache hit rate, errors; per-tenant safe labels.
5) **Billing**: emit usage per query (rows scanned/returned) for cost attribution.
6) **Resilience**: timeouts, retries for DB connections, circuit breakers; graceful degradation when datastore unavailable.
7) **Security**: input sanitization, prepared statements only; audit logging of query params; PII masking if needed.
8) **Testing**: contract tests, load tests for p95/p99 latency, resilience tests for backpressure/timeouts.
9) **Runbooks**: high-latency incidents, cache flush, rate-limit adjustments, ClickHouse/PG failover steps.

## Config to surface
- Auth issuer/audience/scopes; tenant_id claim.
- DB DSN(s), pool sizes, timeouts; read replicas if any.
- Rate limits per tenant; cache settings (TTL, max size); query limits.
- Metrics/tracing exporters; log level/format.
- Billing/usage topic or sink.

## Test coverage checklist
- [ ] Authz + tenant filter enforcement
- [ ] Input validation and limits
- [ ] Query latency/error metrics present
- [ ] Cache hit/miss behavior (if enabled)
- [ ] Billing/usage event emission
- [ ] Resilience: timeout/backoff/circuit tests

## Next suggested steps
- Review handlers and data access layer to replace placeholders above with concrete details; add auth/filter tests and metrics if missing.
