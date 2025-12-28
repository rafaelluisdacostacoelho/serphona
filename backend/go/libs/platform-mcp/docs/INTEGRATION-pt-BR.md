# platform-mcp — Guia rápido de integração (wiring do serviço)

## Stack do executor (ordem recomendada)
Guard (payload/timeout) → Resilient (retry/backoff + circuit) → Cancelable → OutputLimit → Observed (métricas/audit) → Executor base (Static/Streaming).

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

## Métricas
- Contadores/histogramas: `mcp_invocations_total`, `mcp_invocation_latency_seconds` com labels `tenant`, `tool`, `outcome`.
- Labels extras via enrichers: `cache_hit`, `policy_decision`.
- Forneça um registerer Prometheus ou implemente seu próprio `MetricsSink`.

## Auditoria
- Use `AuditObserver` com um `AuditSink` (arquivo/exportador OTEL). Registros redigidos incluem tenant, tool, outcome, duration_ms, request_id, session_id, error_code/message, stage de progresso.

## Fixtures
- Goldens JSON em `protocol/fixtures/*.json` (`invocation_ok`, `invocation_error`, `invocation_cancel`, `invocation_unsupported`).
- Carregados em `protocol/fixtures_external_test.go`; reutilize em testes de contrato/gateway.

## Notas
- `EnsureTenantHeaders` define `X-Tenant-ID` e service ID para chamadas outbound.
- Combine observadores com `NewMultiObserver` para métricas + audit + custom.
- Rate limits e decisões de política podem virar labels via enrichers; escreva contextos com `WithCacheHit` / `WithPolicyDecision` próximos do registry/policy.
