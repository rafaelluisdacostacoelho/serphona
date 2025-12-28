# platform-mcp — Integration quickstart (service wiring)

## Executor stack (recommended order)
Guard (payload/timeout) → Resilient (retry/backoff + circuit) → Cancelable → OutputLimit → Observed (metrics/audit) → Base executor (Static/Streaming).

```go
sink := invoke.NewPrometheusSink(prometheus.DefaultRegisterer)
observer := invoke.NewMetricsObserver(sink, invoke.CacheHitEnricher, invoke.PolicyDecisionEnricher)
observer = invoke.NewMultiObserver(observer, invoke.NewAuditObserver(myAuditSink))

exec := invoke.BuildExecutor(
    invoke.NewStreamingExecutor(handlers),
    invoke.StackConfig{
        Guard:          &invoke.GuardConfig{MaxBodyBytes: 1 << 20, DefaultTimeout: time.Second, MaxTimeout: 5 * time.Second},
        Resilient:      &invoke.ResilientConfig{MaxRetries: 2},
        EnableCancel:   true,
        MaxOutput:      1 << 20,
        LabelEnrichers: []invoke.LabelEnricher{invoke.CacheHitEnricher, invoke.PolicyDecisionEnricher},
        MetricsSink:    sink,
        AuditSink:      myAuditSink,
    },
)

ctx := invoke.WithCacheHit(context.Background(), "true")
ctx = invoke.WithPolicyDecision(ctx, "allow")
_ = exec
```

## Metrics
- Counters/histograms: `mcp_invocations_total`, `mcp_invocation_latency_seconds` with labels `tenant`, `tool`, `outcome`.
- Optional labels via enrichers: `cache_hit`, `policy_decision`.
- Provide a Prometheus registerer or plug your own `MetricsSink`.

## Audit
- Use `AuditObserver` with an `AuditSink` (e.g., writer to file/OTEL exporter). Records are redacted and include tenant, tool, outcome, duration_ms, request_id, session_id, error_code/message, progress stage.

## Fixtures
- JSON goldens live in `protocol/fixtures/*.json` (`invocation_ok`, `invocation_error`, `invocation_cancel`, `invocation_unsupported`).
- Tests load them in `protocol/fixtures_external_test.go`; reuse for contract/gateway tests.

## Notes
- `EnsureTenantHeaders` helper sets `X-Tenant-ID` and optional service ID for outbound calls.
- Compose observers with `NewMultiObserver` to fan out (metrics + audit + custom).
- Rate limits per tool/tenant and policy context can flow via label enrichers; set context values with `WithCacheHit` / `WithPolicyDecision` near the registry/policy layers.
