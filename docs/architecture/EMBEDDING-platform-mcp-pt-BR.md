# Embutindo o platform-mcp (serviços Serphona)

> Público-alvo: engenheiros que vão ligar a lib platform-mcp no serviço platform-mcp, agent-orchestrator e tools-gateway.

## Objetivos
- Garantir isolamento de tenant (headers + policy + RLS) em cada invocação.
- Reaproveitar a pilha compartilhada (guard, retry/circuit, cancel, output cap, rate limit, métricas/audit/tracing).
- Manter envelopes de resposta alinhados ao RESPONSE-ENVELOPE-CONTRACT.

## Wiring mínimo (serviços Go)
1) **Registry**
- Arquivo: `registry.NewFileLoader(path, cacheTTL)`.
- Postgres: `registry.NewPostgresLoader(db, cacheTTL, etagFn)` com escopo de tenant (ver TENANT-RLS-GUIDANCE).
- Cache: `registry.NewCachedLoader(loader, ttl, eTagFn)`.

2) **Policy**
- Comece com `policy.NewMemoryEvaluator(rules)` ou seu avaliador persistido.
- Opcional: `policy.NewMetricsEvaluator` para emitir métricas `mcp_policy_*`.

3) **Pilha de invocação**
```go
exec := invoke.BuildExecutor(
    invoke.NewStaticExecutor(handlers),
    invoke.StackConfig{
        Guard:       &invoke.GuardConfig{MaxBodyBytes: 1 << 20, Timeout: 30 * time.Second},
        Resilient:   &invoke.ResilientConfig{MaxRetries: 2},
        RateLimiter: invoke.NewMemoryRateLimiter(10, 20),
        EnableCancel: true,
        MaxOutput:   1 << 20,
        MetricsSink: promSink, // implementa invoke.MetricsSink (Prometheus incluso)
        AuditSink:   auditSink, // implementa invoke.AuditSink
        AuditOptions: []invoke.AuditOption{invoke.WithAuditSpanEvents(true)},
        LabelEnrichers: []invoke.LabelEnricher{invoke.WithCacheHit(), invoke.WithPolicyDecision()},
        ExtraObservers: []invoke.Observer{invoke.NewTracingObserver("platform-mcp")},
    },
)
```
- Handlers retornam `protocol.InvocationEvent`. Para streaming, use `StreamingExecutor` com cancelamento.
- Sempre valide com `protocol.ValidateInvocation` ou deixe a pilha falhar rápido.

4) **Headers e identidade**
- Use `invoke.EnsureTenantHeaders(reqHeaders, tenantID, requestID, serviceID)` em chamadas para tools.
- Propague `traceparent` e `x-request-id`; o tracing observer já adiciona tenant/tool/request/session.

5) **Envelopes de resposta**
- Use `response.Success(ctx, data, response.WithRequestID(reqID))` e `response.Error(ctx, code, msg, details, response.WithRequestID(reqID))` nos adapters HTTP/gRPC para seguir o RESPONSE-ENVELOPE-CONTRACT.

## Checklist de config (por serviço)
- **Auth**: ISSUER, AUDIENCE, SERVICE_AUDIENCE, TENANT_CLAIM, JWKS_URL/JWT_SECRET.
- **Registry**: POSTGRES_DSN (com RLS), CACHE_TTL/ETAG, fonte de allow-list por tenant.
- **Invocation**: RATE_LIMITS (tenant/tool), MAX_BODY/MAX_OUTPUT, RETRY/BACKOFF/CIRCUIT, TIMEOUT.
- **Observabilidade**: exportador METRICS, exportador TRACE (OTEL), AUDIT_SINK (arquivo/OTEL), SAMPLING (audit + tracing).
- **Headers**: propagar x-request-id + traceparent; garantir que chamadas a tools preservem ambos.

## Testes
- **Policy**: matriz de precedence (allow/deny, env, scopes, rate-limit) com `MemoryEvaluator` ou store.
- **Invocation**: `go test ./invoke -run Resilient|RateLimit|Streaming` para retry/rate/cancel/streaming.
- **Envelope**: `go test ./response`; em handlers HTTP valide status + corpo do envelope.
- **Registry**: cache/ETag com loaders de arquivo/Postgres; inclua fixtures por tenant.
- **Tracing/Audit**: exporters in-memory OTEL para conferir status de span e roteamento/sampling de audit.

## Rollout
- Comece com rate limits conservadores por tenant/tool.
- Habilite sampling em tracing/audit; envie audit para OTEL/log com redação de PII.
- Ajuste retry/circuit por tool; não retente operações não idempotentes.
- Shadow mode: mantenha caminho atual, espelhe para a pilha platform-mcp, compare métricas/audit antes do cutover.

## Referências
- Arquitetura: docs/architecture/README-pt-BR.md
- Envelopes: docs/architecture/RESPONSE-ENVELOPE-CONTRACT-pt-BR.md
- RLS/tenant: docs/architecture/TENANT-RLS-GUIDANCE-pt-BR.md
- Quickstart: docs/INTEGRATION-pt-BR.md
