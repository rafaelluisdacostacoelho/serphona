# platform-mcp — Guia de Embedding e Configuração

Escopo: como embutir a biblioteca platform-mcp no serviço platform-mcp, no agent-orchestrator e no tools-gateway com wiring consistente de política/registry/runtime/observabilidade. Use junto com a matriz de configuração e o checklist de rollout.

## Wiring comum (todos os serviços)
- Registry: priorize loader Postgres com RLS; mantenha fallback de arquivo para dev/cold-start e defina `MCP_TOOLS_ROOT` para isolar leituras de arquivo. Habilite cache TTL/ETag para reduzir churn no DB.
- Policy: aplique allow/deny + rate limits por tenant/tool; default-deny quando não houver política. Publique decisões via `WithPolicyDecision` para rótulos de métricas/audit.
- Runtime de invocação: Guard → Resilient → Cancelable → OutputLimit → Observed → Base (Static/Streaming). Mantenha retries/circuit apenas para ferramentas idempotentes.
- Observabilidade: conecte métricas (`MetricsObserver`), audit (`AuditObserver`), tracing (propague `traceparent`) e enrichers de cache/policy. Exponha `/healthz` e endpoint Prometheus conforme convenção do serviço.
- Headers de saída: use `invoke.EnsureTenantHeaders` para adicionar `x-tenant-id`, `x-request-id`, `traceparent`, `x-service-id` nas chamadas downstream.
- Envelope de resposta: use `response.Success` / `response.Error`; inclua request_id/trace_id conforme o contrato RESPONSE-ENVELOPE.

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

## Serviço platform-mcp
- Onde: handlers HTTP/gRPC que expõem endpoints MCP de invoke/registry.
- Registry: loader Postgres com RLS; habilite cache TTL/ETag; permita fallback de arquivo só em dev com `MCP_TOOLS_ROOT` definido.
- Policy: aplique allow-list/rate limit por tenant; publique decisão de política para métricas/audit.
- Runtime: prefira executor streaming; habilite propagação de cancel para ferramentas longas; limite bytes de saída.
- Observabilidade: inclua tenant/tool nos rótulos de métricas e audit; exporte traces OTEL com request/session IDs.
- Rollout: controle com flags `mcp.enable`, `mcp.shadow.enabled`, `mcp.rate_limit.enabled`, `mcp.retry.enabled`.

## agent-orchestrator
- Onde: chamadas de ferramentas feitas em nome dos agentes.
- Registry: consuma o cache do registry platform-mcp para resolver ferramentas e impor escopo de tenant.
- Policy: valide scopes/roles por ferramenta; injete decisão de política no contexto para observabilidade.
- Runtime: use Resilient + Circuit para ferramentas idempotentes; habilite cancelable/streaming para chamadas longas.
- Observabilidade: propague tenant_id/session_id em traces e audit; use `EnsureTenantHeaders` nas chamadas downstream.

## tools-gateway
- Onde: entrada de clientes externos; envolva handlers com policy/rate-limit + middleware de platform-auth.
- Registry: imponha allow-list por tenant antes de invocar ferramentas; fallback de arquivo apenas em dev isolado.
- Runtime: mantenha Guard e OutputLimit rígidos (clientes não confiáveis); habilite rate-limit por tenant/tool.
- Observabilidade: exponha métricas com rótulos cache_hit/policy_decision; audite todas as invocações de origem externa.

## Checklist rápido de configuração
- Auth: ISSUER, AUDIENCE, SERVICE_AUDIENCE, TENANT_CLAIM, REQUIRED_SCOPES, JWKS_URL/JWT_SECRET.
- Registry: POSTGRES_DSN (RLS), CACHE_TTL/ETAG_FN, ALLOW_LIST_SOURCE, MCP_TOOLS_ROOT (sandbox do loader de arquivo).
- Invocação: MAX_BODY_BYTES, MAX_OUTPUT_BYTES, DEFAULT_TIMEOUT, RETRY/BACKOFF, limites de CIRCUIT, RATE_LIMITS, ENABLE_CANCEL.
- Observabilidade: AUDIT_SINK, METRICS_SINK (Prometheus), exportador de TRACE e sampling, label enrichers.
- Flags de rollout: mcp.enable, mcp.shadow.enabled, mcp.rate_limit.enabled, mcp.retry.enabled, mcp.circuit.enabled, mcp.audit.sampling, mcp.tracing.sampling.

## Validação
- Execute `go test ./...` e `$HOME/go/bin/gosec ./...`.
- Execute `$HOME/go/bin/go-licenses check ./...` e adicione um arquivo LICENSE ou allowlist para módulos internos conforme necessário.
- Smoke test: chame uma ferramenta simples (ex.: echo) com headers de tenant; verifique métricas, audit e envelope de resposta com request_id/trace_id.

## Referências
- Matriz de config: docs/architecture/CONFIG-MATRIX-platform-mcp-pt-BR.md
- Guia rápido de integração: backend/go/libs/platform-mcp/docs/INTEGRATION-pt-BR.md
- Rollout: docs/architecture/ROLLOUT-platform-mcp-pt-BR.md
- Envelope de resposta: docs/architecture/RESPONSE-ENVELOPE-CONTRACT-pt-BR.md
- Guia de RLS/tenant: docs/architecture/TENANT-RLS-GUIDANCE-pt-BR.md
