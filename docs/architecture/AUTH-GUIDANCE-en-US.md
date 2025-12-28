# Auth Guidance — Refresh & Service-to-Service Tokens, CSRF, Fail-Closed (en-US)

## Goals
- Provide opinionated defaults for refresh/service tokens and CSRF/fail-closed behaviors across Serphona services.
- Keep multi-tenant isolation intact (tenant_id everywhere), align with platform-auth contracts.

## Service-to-service (client credentials) tokens
- Audience: issue tokens with a dedicated `SERVICE_AUDIENCE` distinct from user-facing `AUDIENCE`.
- Claims: include `tenantId` (or explicit "platform" tenant for infra-only), `service` (caller ID), `scopes` aligned to allowed internal actions; keep `exp` short (5–15m), `nbf` optional but recommended.
- Issuer: central auth-gateway/issuer; set `iss` consistently.
- Keys: prefer JWKS (`JWKS_URL`, allowed `kid` list), rotate keys; keep HMAC only as fallback for dev.
- Validation: configure platform-auth `AllowedAlgs` to RS256/ES256 for these flows; set `Audience` to `SERVICE_AUDIENCE` per service.
- Propagation: always forward `X-Request-Id`, `Traceparent`, and `X-Tenant-Id`; use client instrumented transport (already propagates request/trace/tenant headers).
- Least privilege: define minimal scopes per service (e.g., `tenant-manager:read-tenants`, `billing:write-invoices`). Reject scope-less internal tokens.

## Refresh tokens (user-facing)
- Storage: httpOnly, Secure cookies; SameSite=Lax by default; rotate on every refresh (issue new refresh+access).
- Audience: user-facing `AUDIENCE`; do not reuse for service-to-service.
- Lifetime: refresh 7–30d; access 5–15m; invalidate on logout/rotation.
- Binding: tie refresh to device/session (`sessionId`), `tenantId`, and optionally IP/UA hash; revoke on mismatch.
- Revocation: keep server-side store (Redis/DB) for refresh family; revoke on logout or suspected compromise.

## CSRF and fail-closed
- CSRF for cookie flows: use double-submit token (`X-CSRF-Token` header matching cookie value) or SameSite=Strict; reject missing/invalid tokens with 403.
- Safe methods: allow GET/HEAD/OPTIONS; require CSRF on state-changing verbs (POST/PUT/PATCH/DELETE).
- Fail-closed defaults: when secret/JWKS/config missing, return 500/401 (not 200); middleware should not panic.
- Header redaction: keep Authorization/Cookie/Proxy-Authorization redacted in logs and safe fields.

## Operational checklist for services
- Configure: `ISSUER`, `AUDIENCE`, `SERVICE_AUDIENCE`, `JWKS_URL`, `JWKS_ALLOWED_KIDS`, `JWT_SECRET` (fallback), `REQUIRED_SCOPES` per route.
- Outbound: use platform-auth HTTP client transport; ensure service name/instance headers are set; rely on auto propagation of request/trace/tenant IDs.
- Inbound: enforce tenant via middleware/helpers; ensure response envelope alignment; expose /healthz + metrics.
- Rotation: monitor JWKS fetch errors; set sensible `JWKS_CACHE_TTL`.

## Rollout plan (service-to-service)
1) Define scope matrix per service (producer/consumer actions).
2) Add `SERVICE_AUDIENCE` config to services; set platform-auth ValidationConfig.Audience accordingly.
3) Enable JWKS validation with allowed kids; roll keys; drop HMAC for internal flows once JWKS is live.
4) Instrument outbound calls to use propagated headers; add tests for tenant/request ID forwarding.

## Rollout plan (refresh/CSRF)
1) Enforce Secure+httpOnly cookies; SameSite=Lax (Strict if acceptable).
2) Implement double-submit or equivalent CSRF token on state-changing routes.
3) Rotate refresh on each use; store family IDs for revocation; add logout revocation.
4) Add tests: missing/invalid CSRF -> 403; missing secret/JWKS -> 500/401 fail-closed.
