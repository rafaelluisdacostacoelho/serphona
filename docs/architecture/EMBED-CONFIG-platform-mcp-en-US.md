# platform-mcp — Embedding & Configuration Guide

Scope: how to embed the platform-mcp library into platform-mcp (service), agent-orchestrator, and tools-gateway with consistent policy/registry/runtime/observability wiring. Use with the config matrix and rollout checklist.

## Common wiring (all services)
- Registry: prefer Postgres loader with RLS; keep file fallback for dev/cold-start and set `MCP_TOOLS_ROOT` to sandbox file reads. Enable cache TTL/ETag to reduce DB churn.
- Policy: enforce allow/deny + rate limits per tenant/tool; default-deny when policy is missing. Surface decisions via `WithPolicyDecision` for metrics/audit labels.
- Invocation stack: Guard → Resilient → Cancelable → OutputLimit → Observed → Base (Static/Streaming). Keep retries/circuit only for idempotent tools.
- Observability: wire metrics (`MetricsObserver`), audit (`AuditObserver`), tracing (propagate `traceparent`), and cache/policy enrichers. Expose `/healthz` and Prometheus scrape endpoints per service convention.
- Outbound headers: use `invoke.EnsureTenantHeaders` to add `x-tenant-id`, `x-request-id`, `traceparent`, `x-service-id` to downstream tool calls.
- Response envelope: use `response.Success` / `response.Error` helpers; include request_id/trace_id per RESPONSE-ENVELOPE contract.

```go
sink := invoke.NewPrometheusSink(prometheus.DefaultRegisterer)
audit := invoke.NewAuditObserver(auditSink)
metrics := invoke.NewMetricsObserver(sink, invoke.CacheHitEnricher, invoke.PolicyDecisionEnricher)
observer := invoke.NewMultiObserver(metrics, audit)

exec := invoke.BuildExecutor(
    invoke.NewStreamingExecutor(handlers),
    invoke.StackConfig{
        Guard:        &invoke.GuardConfig{MaxBodyBytes: 1 << 20, DefaultTimeout: time.Second, MaxTimeout: 5 * time.Second},
        Resilient:    &invoke.ResilientConfig{MaxRetries: 2},
        EnableCancel: true,
        MaxOutput:    1 << 20,
        MetricsSink:  sink,
        AuditSink:    auditSink,
        LabelEnrichers: []invoke.LabelEnricher{
            invoke.CacheHitEnricher, invoke.PolicyDecisionEnricher,
        },
    },
)
```

## platform-mcp service
- Where: HTTP/gRPC handlers that expose MCP invoke/registry endpoints.
- Registry: Postgres loader with RLS; enable cache TTL/ETag; allow file fallback only in dev with `MCP_TOOLS_ROOT` set.
- Policy: enforce per-tenant allow-list/rate limit; emit policy decision labels for audit/metrics.
- Runtime: prefer streaming executor; enable cancel propagation for long-running tools; cap output bytes.
- Observability: include tenant/tool labels on metrics and audit; export OTEL traces with request/session IDs.
- Rollout: gate with `mcp.enable`, `mcp.shadow.enabled`, `mcp.rate_limit.enabled`, `mcp.retry.enabled` flags.

## agent-orchestrator
- Where: outbound tool calls made on behalf of agents.
- Registry: consume platform-mcp registry cache to resolve tools and enforce tenant scoping.
- Policy: validate scopes/roles per tool; pass policy decision into context for observability.
- Runtime: use Resilient + Circuit for idempotent tools; enable cancelable/streaming for long calls.
- Observability: propagate tenant_id/session_id into traces and audit; ensure `EnsureTenantHeaders` is used for downstream calls.

## tools-gateway
- Where: ingress for external tools; wrap handlers with policy/rate-limit + platform-auth middleware.
- Registry: enforce tenant allow-list before invoking tools; file fallback only in isolated dev.
- Runtime: keep Guard and OutputLimit strict (untrusted clients); enable rate-limit per tenant/tool.
- Observability: expose metrics with cache_hit/policy_decision labels; audit all externally sourced invocations.

## Configuration quick checklist
- Auth: ISSUER, AUDIENCE, SERVICE_AUDIENCE, TENANT_CLAIM, REQUIRED_SCOPES, JWKS_URL/JWT_SECRET.
- Registry: POSTGRES_DSN (RLS), CACHE_TTL/ETAG_FN, ALLOW_LIST_SOURCE, MCP_TOOLS_ROOT (sandbox file loader).
- Invocation: MAX_BODY_BYTES, MAX_OUTPUT_BYTES, DEFAULT_TIMEOUT, RETRY/BACKOFF, CIRCUIT thresholds, RATE_LIMITS, ENABLE_CANCEL.
- Observability: AUDIT_SINK, METRICS_SINK (Prometheus), TRACE exporter and sampling, label enrichers.
- Rollout flags: mcp.enable, mcp.shadow.enabled, mcp.rate_limit.enabled, mcp.retry.enabled, mcp.circuit.enabled, mcp.audit.sampling, mcp.tracing.sampling.

## Validation
- Run `go test ./...` and `$HOME/go/bin/gosec ./...`.
- Run `$HOME/go/bin/go-licenses check ./...` and add a LICENSE file or allowlist as needed for internal modules.
- Smoke test: call a simple tool (e.g., echo) with tenant headers; verify metrics, audit, and response envelope contain request_id/trace_id.

## References
- Config matrix: docs/architecture/CONFIG-MATRIX-platform-mcp-en-US.md
- Integration quickstart: backend/go/libs/platform-mcp/docs/INTEGRATION-en-US.md
- Rollout: docs/architecture/ROLLOUT-platform-mcp-en-US.md
- Response envelope: docs/architecture/RESPONSE-ENVELOPE-CONTRACT-en-US.md
- Tenant RLS guidance: docs/architecture/TENANT-RLS-GUIDANCE-en-US.md
