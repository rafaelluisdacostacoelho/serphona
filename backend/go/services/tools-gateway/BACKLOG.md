# Tools Gateway — Backlog (en-US)

## Status snapshot
- Audit complete: Gin HTTP only; mock auth middleware injects random tenant/user UUIDs (no JWT/scopes). Tool CRUD, integration CRUD, OAuth flows, and execution path exist; tenant scoping is unchecked and configs lack validation. No rate limits, quotas, billing hooks, or observability beyond defaults. DB connections use env DATABASE_URL without migrations or TLS. Execution uses HTTP client with schema validation but no retries/backoff; credits consumed stored but not billed.

## Audit findings
- Auth/tenant: no real auth middleware; tenant_id/user_id are random per request; no scope/role checks; integrations and tools not filtered by authenticated tenant except by context when present; CORS/rate limits absent.
- Validation/schema: input/output schemas checked via validator, but schema validity only shallow (IsValidSchema) and no size/field limits; ExecuteTool trusts tenant context and allows any tool ID across tenants unless tenant-tool mapping is configured; lacks payload size caps.
- Execution/resilience: HTTP client executes downstream without retries/backoff/circuit breaker; timeout fixed at 30s; errors return 200 with error body; no DLQ/idempotency; no per-tool concurrency caps.
- Data layer: GORM with DATABASE_URL default; migrations not invoked; tables and uniqueness constraints assumed; no RLS/tenant enforcement in queries (tenant filter optional). No TLS/pool tuning.
- Observability: no metrics, tracing, or structured logging in handlers/usecases; health only. No audit logs for tool executions or OAuth.
- Billing/quotas: credits consumed recorded but no billing event or quota enforcement per tenant/tool/user; no rate limits.
- OAuth/integrations: OAuth service supports state/PKCE/token refresh but no persistence validation or replay protection checks; provider allowlist absent; token revocation best-effort; tokens stored without encryption at rest.
- Security: Mock auth leaves endpoints open; no input size limits; downstream URLs from tool definitions not validated/allowlisted; secrets (client secret, API keys) not masked; HTTPS/TLS not exposed.
- MCP/Agent alignment: no MCP interface/catalog; execution path is HTTP-only; no gRPC.

## Action items
1) Auth & tenant: add real JWT middleware with issuer/audience/scopes, enforce tenant scoping on tools/integrations/executions; drop mock IDs; add role-based access and CORS/rate limits.
2) Validation & safety: enforce payload size limits, schema depth/field limits, allowed outbound host/protocol allowlist, and per-tool/tenant enablement checks; cap TopK/limits if present.
3) Resilience: add retries/backoff and circuit breaker for downstream HTTP; timeouts per tool; idempotency for executions; DLQ or error queue for failed executions; clearer error codes (use 4xx/5xx, not 200 on failure).
4) Data/migrations: wire migrations on startup; add tenant scoping and indexes; ensure unique constraints (tool name, tenant-tool). Add DB TLS/pool config.
5) Observability/audit: add Prometheus metrics (latency, errors, rate-limit blocks, retries), tracing (HTTP inbound + outbound), and structured audit logs for tool exec and OAuth events.
6) Billing/quotas: enforce per-tenant/tool quotas and rate limits; emit usage events (tool, latency, status, credits) to billing/analytics sink; define pricing model for credits.
7) OAuth/integrations: enforce provider allowlist, secure state storage/expiry, token encryption at rest, revocation callbacks, and per-tenant integration isolation; mask secrets in logs.
8) MCP/Agent: expose catalog/discovery endpoint or gRPC for agent-orchestrator/MCP; standardize tool schema contracts and versioning.
9) Testing: contract tests for tool schema validation, auth/tenant isolation, execution error paths, OAuth flows with fake provider, retry/backoff behavior, and metrics/tracing emission.
10) Runbooks: tool onboarding/disable, rotating secrets, handling downstream outages/DLQ replay, quota/rate-limit tuning, OAuth token refresh failures.
11) Response contract: adopt the shared envelope helper (success data/meta; error code/message/details/trace_id) for all HTTP handlers and adjust execution error mapping/tests accordingly.

## Config to surface
- HTTP addr, timeouts, body size, CORS, rate limits.
- Auth issuer/audience/JWKS, required scopes/roles; tenant claim name.
- DATABASE_URL with TLS/pool/migrations toggle.
- Downstream execution settings per tool: allowed hosts, timeouts, retries/backoff, circuit breaker, max concurrency.
- OAuth: providers allowlist, client IDs/secrets, state TTL storage, token encryption, redirect URLs.
- Observability exporters (Prometheus/Otel), log level/format; billing/usage event sink.

## Test coverage checklist
- [ ] JWT auth + tenant isolation on tool/integration CRUD and execute
- [ ] Schema validation and payload size/host allowlist enforcement
- [ ] Execution retries/backoff and error/status mapping (non-200s)
- [ ] Quotas/rate limits and billing usage events
- [ ] OAuth flows (state TTL/PKCE, token refresh/revoke)
- [ ] Metrics/tracing/audit emitted and migrations applied

## Next suggested steps
- Replace mock auth with platform-auth middleware and tenant scoping; add validated config (limits, allowlists) and migrations. Then implement retries/backoff, metrics/tracing, quotas/billing events, and secure OAuth/integration handling before MCP integration.

## Referências
- Este serviço depende do [BACKLOG-AUTH.md](../../../BACKLOG-AUTH.md) para alinhamento com autenticação e autorização multi-tenant.
