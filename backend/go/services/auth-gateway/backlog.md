# Auth Gateway — Backlog (en-US)

## Status snapshot
- Audit complete: Gin service with local Postgres via GORM, HS256 tokens using static secret, no audience/issuer validation, no key rotation, no rate limiting or brute-force controls. Tenant creation is stubbed (random UUID, no tenant-manager call). OAuth Google/Microsoft validated via OIDC, Apple implementation does not verify JWT properly. Redis config unused. No tracing/metrics, no structured audit trail, permissive CORS, no TLS, and configs lack validation.

## Audit findings
- Auth/token model: HS256 single secret from env defaulting to a dev string; issuer fixed to "serphona-auth", audience not enforced, no clock skew handling, no key rotation, no JWKS, and refresh/session expiry hardcoded to 7d (not using configured refresh duration). No scope/permissions in claims beyond role string.
- Session handling: refresh tokens stored in Postgres, but session expiry is fixed; cleanup/revocation relies on DB but no background job; logout revokes all sessions; refresh revokes only the used token. No device/IP binding checks.
- Tenant: tenant service is a stub returning uuid.New() and just prints; no call to tenant-manager, no namespace/ACL alignment, and no validation of tenant limits. Tokens trust stored tenant ID; middleware only checks signature, not tenancy access to downstream resources.
- OAuth: Google/Microsoft use OIDC verifier; Apple path fetches userinfo with bearer token and skips JWT verification; redirect URLs come straight from env without allowlist/PKCE; state stored but no cleanup scheduler (only manual cleanup helper).
- HTTP/API: Gin default with permissive CORS *, no rate limits, no request size caps, no password reset/verification flows, no MFA, no lockout on failed logins; password min length only. Health is static.
- Data layer: GORM auto-migrate at startup, soft deletes, no migrations safety; SSL disabled by default; no connection pool/tuning beyond basic max open/idle; no RLS for tenant isolation.
- Observability: zap logger exists but handlers only log internal errors; no structured audit log for auth events; no tracing, metrics, or correlation IDs.
- Security/compliance: no TLS termination options, no secret management (loads .env), no PII masking, CORS allows credentials with wildcard origin, no CSRF protections for cookies (though tokens are JSON), no input allowlists for redirect URIs/providers.

## Action items
1) Token security: add issuer/audience validation, clock skew leeway, configurable signing algorithm with rotation (prefer asymmetric + JWKS); derive access/refresh TTLs from config; include scopes/permissions and jti; add refresh token rotation and reuse detection.
2) Tenant/RLS: integrate with tenant-manager API, validate tenant status/limits, and ensure tokens carry authoritative tenant claims; add middleware to enforce tenant context on downstream calls; consider per-tenant session quotas.
3) OAuth hardening: verify Apple tokens properly (JWT + JWKS), require provider allowlist and enforce PKCE/state TTL cleanup (background job); restrict redirect URIs to configured allowlist; map verified email and provider IDs safely.
4) Registration/login security: add rate limiting and lockout/backoff for login/register/refresh; password policy (entropy/history), email verification flow, optional MFA; device/IP binding for refresh tokens; add password reset.
5) API/HTTP: tighten CORS (explicit origins, disable credentials unless needed), set HTTP/idle timeouts, request body/size limits, optional TLS/mTLS; add versioned OpenAPI and error contract.
6) Data layer: use migrations instead of auto-migrate; enable SSL/TLS to Postgres; add connection settings in config; avoid storing plaintext refresh tokens or hash them; scheduled cleanup for sessions/oauth states.
7) Observability/audit: add structured audit log for auth events (login success/fail, register, refresh, logout, OAuth link), tracing (HTTP + DB + external IdP), Prometheus metrics (latency, error rates, rate-limit blocks), and correlation/request IDs in logs.
8) Resilience: add retries/backoff for IdP and tenant-manager calls, circuit breaker, and fail-closed defaults; health/ready endpoints should check DB and external deps.
9) Billing/usage: emit events for MAU or token issuance per tenant for billing/analytics sinks.
10) Docs/runbooks: document config matrix, key rotation steps, incident playbooks for auth outage/token compromise, and OAuth redirect management.
11) Response contract: adopt the shared envelope helper (success data/meta; error code/message/details/trace_id) once provided by platform-auth/core and migrate handlers/tests accordingly.

## Config to surface
- Server host/port, timeouts, TLS/mTLS settings, allowed origins/redirect URIs.
- Postgres DSN (with SSL/TLS), pool sizes, migrations toggle.
- JWT signing keys/JWKS endpoint, issuer, audience, leeway, access/refresh TTLs, rotation interval, required scopes/roles.
- OAuth providers: enabled flag, client IDs/secrets, redirect URLs/allowlist, PKCE requirement.
- Tenant-manager base URL and auth, per-tenant quotas.
- Rate limits per IP/tenant/client for login/register/refresh; lockout/backoff thresholds.
- Metrics/tracing exporters and audit sink (e.g., Kafka/ClickHouse).

## Test coverage checklist
- [ ] Token validation (issuer/audience/expiry/leeway) + rotation/JWKS
- [ ] Tenant propagation and cross-tenant isolation in tokens and middleware
- [ ] Rate limiting/lockout on login/register/refresh; brute-force defenses
- [ ] OAuth flows (state TTL, PKCE, Apple JWT verification, redirect allowlist)
- [ ] Session refresh rotation/reuse detection and cleanup scheduler
- [ ] Metrics/tracing/audit emission on auth events
- [ ] TLS/DB config validation and migrations

## Next suggested steps
- Implement secure config (validated) and JWT/JWKS rotation with issuer/audience enforcement; replace tenant stub with tenant-manager integration and tighten CORS/rate limits. Add OAuth hardening (PKCE/state cleanup, Apple JWT verify), observability/audit, and migration-based DB setup before rolling out.
