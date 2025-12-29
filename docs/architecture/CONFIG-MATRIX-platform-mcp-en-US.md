# Platform-mcp Configuration Matrix (Services: platform-mcp, agent-orchestrator, tools-gateway)

> Purpose: align service configs per concern (auth, registry, invocation, observability, rollout) and keep parity across environments.

## Auth / Identity
- ISSUER / AUDIENCE / SERVICE_AUDIENCE: must match platform-auth JWKS issuer and service audiences.
- TENANT_CLAIM: claim carrying tenant_id (required for RLS and metrics labels).
- REQUIRED_SCOPES: per tool/route scopes; keep in sync with policy rules.
- JWKS_URL or JWT_SECRET: per env; prefer JWKS + caching; rotate secrets safely.
- HEADERS: propagate `x-request-id`, `traceparent`, `authorization` (inbound only). Do not forward cookies.

## Registry
- POSTGRES_DSN: RLS enabled; per-tenant policies applied (see TENANT-RLS-GUIDANCE).
- CACHE_TTL / ETAG_FN: align with registry change cadence; shorter TTL in dev.
- ALLOW_LIST_SOURCE: per-tenant tool allow-list (DB table or file).
- FALLBACK_LOADER: file loader for cold-start/dev; ensure tenant filtering.
- MCP_TOOLS_ROOT: required when enabling file fallback; sandbox file reads to a fixed root to avoid path traversal.

## Policy / RBAC
- EVALUATOR: Memory for tests/dev; DB-backed in prod.
- RATE_LIMIT_POLICY: per tenant/tool rule set; default deny on missing policy.
- METRICS: enable `mcp_policy_*` with `MetricsEvaluator`; label env/tenant/tool/rule.

## Invocation Runtime
- MAX_BODY_BYTES / MAX_OUTPUT_BYTES: cap inputs/outputs; align with tool constraints.
- TIMEOUT: default per tool; circuit breaker thresholds per dependency.
- RETRY/BACKOFF: idempotent tools only; exponential backoff defaults (50ms<<attempt).
- CIRCUIT_BREAKER: threshold/reset per tool; start conservative.
- RATE_LIMITS: token bucket per tenant/tool; tune burst for UX.
- ENABLE_CANCEL: true for streaming tools; ensure downstream honors context.

## Headers / Outbound
- Ensure `EnsureTenantHeaders` adds `x-tenant-id`, `x-request-id`, `traceparent`, `x-service-id` to downstream tool calls.
- Enforce lowercase header propagation in gateways.
- Strip sensitive headers when crossing trust boundaries.

## Observability / Audit
- METRICS: Prometheus sink; scrape endpoints exposed by services.
- TRACING: OTEL exporter (OTLP/gRPC); tracer name `platform-mcp`; sampling per env.
- AUDIT: sink (file/OTEL); enable sampling/router; redact PII; include tenant/tool/request/session IDs.
- LABEL ENRICHERS: cache_hit, policy_decision.

## Response Envelope
- Use `response.Success` / `response.Error` helpers; include request_id and trace_id.
- HTTP: follow RESPONSE-ENVELOPE-CONTRACT; gRPC map metadata traceparent/x-request-id.
- Fuzz tests: header parsing for x-request-id/traceparent to avoid injection.

## Rollout / Ops
- Feature flags: rate-limit on/off per tool; audit sampling rate; tracing sampling.
- Shadow mode: mirror calls through platform-mcp stack, compare audit/metrics before cutover.
- Dashboards: calls/latency/error by tenant/tool; rate-limit denials; circuit breaker opens; audit volume.
- Alerts: circuit opens, sustained 4xx/5xx per tenant/tool, audit sink failures.

## Environment Parity Checklist
- ✅ Dev: memory policy, file registry fallback, local OTEL collector, permissive rate limits.
- ✅ Staging: Postgres registry with RLS, policy DB, OTEL exporter to staging, audit sampling enabled.
- ✅ Prod: strict RLS, policy DB required, conservative rate/circuit, audit to durable sink, sampling tuned.
