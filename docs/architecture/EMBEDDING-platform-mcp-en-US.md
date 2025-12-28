# Embedding platform-mcp (Serphona services)

> Target audience: engineers wiring platform-mcp library into platform-mcp service, agent-orchestrator, and tools-gateway.

## Goals
- Enforce tenant isolation (headers + policy + RLS contracts) on every invocation.
- Reuse shared invocation stack (guard, retry/circuit, cancel, output caps, rate limit, metrics/audit/tracing).
- Keep response envelopes consistent with RESPONSE-ENVELOPE-CONTRACT.

## Minimal wiring (Go services)
1) **Load registry**
- File: use `registry.NewFileLoader(path, cacheTTL)`.
- Postgres: use `registry.NewPostgresLoader(db, cacheTTL, etagFn)` with tenant scoping (see TENANT-RLS-GUIDANCE).
- Wrap with cache: `registry.NewCachedLoader(loader, ttl, eTagFn)`.

2) **Policy**
- Start with `policy.NewMemoryEvaluator(rules)` or your persistence-backed evaluator.
- Optionally decorate with `policy.NewMetricsEvaluator` to emit `mcp_policy_*` metrics.

3) **Invocation stack**
```go
exec := invoke.BuildExecutor(
    invoke.NewStaticExecutor(handlers),
    invoke.StackConfig{
        Guard:       &invoke.GuardConfig{MaxBodyBytes: 1 << 20, Timeout: 30 * time.Second},
        Resilient:   &invoke.ResilientConfig{MaxRetries: 2},
        RateLimiter: invoke.NewMemoryRateLimiter(10, 20),
        EnableCancel: true,
        MaxOutput:   1 << 20,
        MetricsSink: promSink, // implement invoke.MetricsSink (Prometheus provided)
        AuditSink:   auditSink, // implement invoke.AuditSink
        AuditOptions: []invoke.AuditOption{invoke.WithAuditSpanEvents(true)},
        LabelEnrichers: []invoke.LabelEnricher{invoke.WithCacheHit(), invoke.WithPolicyDecision()},
        ExtraObservers: []invoke.Observer{invoke.NewTracingObserver("platform-mcp")},
    },
)
```
- Handlers return `protocol.InvocationEvent`. For streaming, use `StreamingExecutor` with cancellation awareness.
- Always validate requests via `protocol.ValidateInvocation` or let the stack fail fast.

4) **Headers and identity**
- Use `invoke.EnsureTenantHeaders(reqHeaders, tenantID, requestID, serviceID)` for outbound calls to tools.
- Carry `traceparent` and `x-request-id` across hops; tracing observer already tags tenant/tool/request/session IDs.

5) **Response envelopes**
- Use `response.Success(ctx, data, response.WithRequestID(reqID))` and `response.Error(ctx, code, msg, details, response.WithRequestID(reqID))` for HTTP/gRPC adapters to stay aligned with RESPONSE-ENVELOPE-CONTRACT.

## Config checklist (per service)
- **Auth**: ISSUER, AUDIENCE, SERVICE_AUDIENCE, TENANT_CLAIM, JWKS_URL/JWT_SECRET.
- **Registry**: POSTGRES_DSN (with RLS), CACHE_TTL/ETAG, tenant allow-list source.
- **Invocation**: RATE_LIMITS (tenant/tool), MAX_BODY/MAX_OUTPUT, RETRY/BACKOFF/CIRCUIT, TIMEOUT.
- **Observability**: METRICS exporter, TRACE exporter (OTEL), AUDIT_SINK (file/OTEL), SAMPLING (audit + tracing).
- **Headers**: x-request-id + traceparent propagation; ensure downstream tool calls preserve them.

## Testing guidance
- **Policy**: table-driven precedence tests (allow/deny, env, scopes, rate-limit) using `MemoryEvaluator` or your store.
- **Invocation**: `go test ./invoke -run Resilient|RateLimit|Streaming` for retry/rate/cancel/streaming paths.
- **Envelope**: `go test ./response` for shape; add handler-level tests asserting HTTP status + envelope body.
- **Registry**: cache/ETag tests with file/Postgres loaders; include tenant-scoped fixtures.
- **Tracing/Audit**: OTEL in-memory exporters to assert span status/events and audit routing/sampling.

## Production rollout tips
- Start with conservative rate limits per tenant/tool.
- Enable tracing/audit sampling; send audit to OTEL/log sink with PII redaction.
- Circuit breaker and retry tuned per tool; avoid retrying non-idempotent tools.
- Shadow mode: keep existing path, mirror to platform-mcp stack, compare audit/metrics before cutover.

## References
- Architecture: docs/architecture/README-en-US.md
- Response envelope: docs/architecture/RESPONSE-ENVELOPE-CONTRACT-en-US.md
- RLS/tenant: docs/architecture/TENANT-RLS-GUIDANCE-en-US.md
- Integration quickstart: docs/INTEGRATION-en-US.md
