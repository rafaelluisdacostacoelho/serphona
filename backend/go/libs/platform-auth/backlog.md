# platform-auth — Backlog (en-US)

## Status snapshot
- Audit complete: library is minimal HMAC-only JWT validation with Gin middleware; no issuer/audience/clock-skew checks, no JWKS/rotation, no scopes, no gRPC/HTTP client hardening, no metrics/tracing/tenant RLS helpers, and docs/config are thin.

## Audit findings
- JWT validation: only HMAC secret, no issuer/audience validation, no clock skew leeway, no key rotation/JWKS; errors collapsed to generic invalid/expired; token size not bounded; no kid/alg allow-list; refresh flows not addressed.
- Claims: requires UUID user/tenant and role enum, but does not enforce exp/nbf manually; no scopes/permissions; no service-to-service claims pattern.
- Middleware (Gin): hard fail/panic if secret missing; no context propagation for trace/request ID; no per-route scopes; no CORS/body-size limits; only Gin (no gRPC/chi/net-http); responses lack error codes consistency.
- Client: static baseURL, no retries/backoff, no context or timeout per call, no TLS/custom transport, no structured errors; no authz helper for internal calls; no service identity.
- RLS/tenant: no helper to enforce tenant in DB queries or Kafka headers; no guard against cross-tenant access beyond JWT claim presence.
- Observability: no Prometheus/OTEL metrics for auth successes/failures/latency; no trace attributes for tenant/user; logs can leak secrets.
- Security: no max token length, no header hardening, no CSRF guidance for cookies, no algorithm allow-list, no audience/issuer enforcement.
- Docs: missing config matrix (issuer, audience, jwks, scopes), multi-framework examples, and upgrade/breaking-change guidance.

## Action items
1) Token validation: add issuer/audience/clock-skew enforcement, alg/kid allow-list, max token length, scopes/permissions support, optional JWKS with rotation and cache, leeway controls, refresh-token guidance; support service-to-service tokens (client credentials) with separate audience.
2) Middleware/interceptors: provide Gin/chi/net-http and gRPC interceptors with per-route scopes, tenant + trace/request IDs propagation, panic recovery, optional CORS/body-size limits, and consistent error payloads.
3) Client: add context-aware methods, retries/backoff with circuit breaker, TLS config, service identity headers, structured errors; support mTLS when configured.
4) Tenant/RLS helpers: helpers to inject tenant_id into DB contexts/queries and Kafka headers; guard rails for cross-tenant enforcement; guidance for pgvector/ClickHouse.
5) Observability: emit metrics for auth success/failure/latency, token type, and reasons; OTEL spans/attributes for tenant/user/scope; structured logging without secret leakage.
6) Security hardening: algorithm allow-list, token length cap, header parsing strictness, CSRF guidance for cookie-based flows, fail-closed defaults; secret load should not panic in middleware path.
7) Docs/examples: expand README/guide with config table (ISSUER, AUDIENCE, JWKS_URL/PEM, REQUIRED_SCOPES, TENANT_CLAIM, CLOCK_SKEW, AUTH_GATEWAY_URL, JWT_SECRET), multi-framework examples (Gin/chi/gRPC), and client usage with retries/mTLS; add RLS and Kafka header guidance.
8) Governance: versioning/breaking-change notes for claims format and middleware behaviors; deprecation path for any API changes.
9) Testing/benchmarks: golden tokens (valid/expired/wrong alg/wrong aud/issuer/missing tenant/scope), JWKS rotation tests, middleware scope/role tests, client retry tests, fuzz for header parsing, benchmark middleware overhead.

## Config to surface
- JWT_SECRET (fallback), JWKS_URL/JWKS_CACHE_TTL, ISSUER, AUDIENCE, SERVICE_AUDIENCE, REQUIRED_SCOPES, TENANT_CLAIM, CLOCK_SKEW, MAX_TOKEN_BYTES, AUTH_GATEWAY_URL, TLS_CA/CERT/KEY for client, CORS settings, TRACE/REQUEST ID header names.

## Test coverage checklist
- [ ] Valid token acceptance (HMAC + JWKS)
- [ ] Invalid/expired/nbf/issuer/audience/alg/kid errors
- [ ] Missing scope/tenant rejection and per-route scope enforcement
- [ ] Middleware context injection (tenant/user/trace/request IDs) across Gin/chi/gRPC
- [ ] Client retries/backoff and TLS/mTLS paths
- [ ] Metrics/tracing emitted without leaking secrets
- [ ] Benchmarks for middleware overhead

## Next steps
- Add issuer/audience/clock-skew validation and optional JWKS with rotation; then implement per-route scopes and multi-framework middleware/interceptors with observability hooks; follow with client retries/TLS and RLS helpers; update docs/config tables and add golden token test suite.
