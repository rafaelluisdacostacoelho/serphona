# Tenant Manager — Backlog (en-US)

## Status snapshot
- Audit complete: HTTP + gRPC service with JWT middleware (HS256/optional RS256); tenant routes now enforce read/write scopes but roles remain unused. Tenants and API keys exist with validation; Kafka events optional (noop by default); Redis cache optional. No rate limiting, no RLS/multitenant DB guarantees, and quotas not enforced despite interfaces. Metrics endpoint is static gauge; tracing absent. TLS, CORS tightening, and config validation are minimal; password masking TODO.
- Tenant safeguards tightened: repos/handlers enforce tenant context, DELETE honors platform override, Kafka producer injects X-Tenant-Id via helper with table-driven tests, and event publisher now retries with configurable backoff and optional DLQ fallback. HTTP hardened with body limits, rate limiting, and configurable CORS.

## Audit findings
- Authz: JWT checked for tenant_id/sub but no scopes/roles; same secret for HTTP/gRPC; no mTLS; CORS middleware likely permissive; no rate limits or per-tenant quotas on API/gRPC.
- Tenant lifecycle: create/list/update/delete implemented; plan change adjusts limits in code but not persisted/enforced elsewhere; slug uniqueness handled; status transitions allowed but no audit trail; delete is soft but no cascading cleanup (API keys, cache, events).
- API keys: domain supports permissions, IP allowlist, revoke/expire; gRPC/HTTP exposure uses app service; events publisher TODO (nil), so no audit/usage events; cache may be nil; no brute-force protection or key prefix length limits documented.
- Quotas/usage: repository interface defines quota/usage but HTTP/gRPC handlers and services do not enforce or update usage; no linkage to billing or rate limits.
- Data layer: Postgres migrations optional; no RLS policies, tenant scoping depends on app logic; password masking TODO in logs; no TLS settings for DB/Redis/Kafka. Auto-migrate toggle but no schema validation.
- Eventing: Kafka optional; when unavailable a noop publisher is used silently; no retries/backoff/idempotency for tenant events.
- Observability: zap logger via middleware; metrics endpoint hardcoded gauge; no Prometheus metrics for ops; no tracing. Health/ready check DB/Redis only via handler; gRPC health set SERVING statically.
- Security: no request size limits beyond Gin defaults; no allowed origins; JWT audience parsing but ParseJWT details unknown; API docs served openly; no audit logging for admin actions.
- Resilience: graceful shutdown present; Redis/Kafka optional; no backoff/retry around DB/Kafka; cache invalidation best-effort; rate limits missing.

## Action items
1) AuthN/Z: scopes enforced on HTTP/gRPC tenant routes; next: roles where needed, rate limits per tenant/IP, request size limits; tighten CORS; support mTLS/TLS configs.
2) Tenant/quotas: implement quota persistence/enforcement (ingest/query/telephony/etc.), usage counters, and align with billing-service; add RLS or strict tenant scoping in repos/queries.
3) API keys: wire event publisher for create/revoke/use, add brute-force/lockout/IP throttle, and document/enforce permission model; expose hashed storage and prefix length limits; add audit logs.
4) Events/resilience: add Kafka retries/backoff and DLQ for tenant events; fail loud when publisher disabled in prod; add idempotency keys for create/update to avoid duplicates.
5) Data & migrations: require migrations on startup or fail; add DB/Redis TLS settings; implement password masking in logs; validate config (required secrets, URLs, pool limits) before start.
6) Observability: add Prometheus metrics (latency, errors, cache hits, event publish outcomes), tracing for HTTP/gRPC/DB/Kafka, and structured audit logging (who/what/tenant).
7) Cleanup & lifecycle: cascade or enqueue cleanup on tenant delete (API keys, cache, events, related services); add suspension reasons and audit trail; add plan change events for billing.
8) Testing: API/gRPC contract tests, authz/scope tests, quota enforcement, API key auth happy/negative, event emission with Kafka fake, cache invalidation, and RLS/leakage tests.
9) Runbooks: tenant deletion/restore, quota override, Kafka outage handling, key rotation/secret rotation, and recovery from failed migrations.
10) Response contract: adopt the shared envelope helper (success data/meta; error code/message/details/trace_id) for HTTP/Gin handlers and align gRPC status/metadata once available.

## Config to surface
- JWT issuer/audience, HS/RS keys, required scopes/roles; CORS allowed origins; rate limits and max body size.
- DB URL, pool sizes, TLS, migrations toggle/path; Redis URL/TTL/TLS; Kafka brokers/topic prefix/retries/DLQ.
- Quota defaults and plan limits; billing integration endpoints; event flags.
- Metrics/tracing exporters, log level/format, audit sink.

## Test coverage checklist
- [ ] Scope/role enforcement on HTTP/gRPC (positive/negative, tenant vs platform contexts) — scopes covered; roles pending
- [ ] Quota enforcement and RLS/tenant isolation
- [ ] API key auth (success/failure), rate limits, and event emission
- [ ] Kafka publish retry/DLQ paths and cache invalidation (include tenant header propagation) — retry/backoff+DLQ implemented and covered; cache invalidation pending
- [ ] Rate limits per tenant/IP and tighter CORS — IP rate limit and configurable CORS in place; per-tenant limits pending
- [ ] Metrics/tracing/audit emitted and config validation errors

### Coverage notes
- Tenant context enforcement covered in handlers (including DELETE) and repos; Kafka producer has table-driven tenant header helper tests.
- Scopes enforced and tested on HTTP (missing-scope 403) and gRPC (PermissionDenied when scopes absent).
- Kafka publisher retry/backoff + DLQ fallback covered with producer stub tests.
- HTTP hardening: body size limit, IP rate limit, and configurable CORS wired at startup.

## Next suggested steps
- Add validated config and stricter authz/rate limits; implement quotas and tenant-scoped RLS, wire API key/tenant events with retries/DLQ, and add metrics/tracing plus audit logging.
