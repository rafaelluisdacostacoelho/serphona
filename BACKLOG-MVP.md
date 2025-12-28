# MVP Implementation Backlog (Copilot Guide)

Ordem sugerida para liberar o MVP do Serphona com foco em MCP, catálogo e execução de ferramentas.

## 1) platform-mcp (serviço novo)
- Scaffold HTTP+gRPC com platform-auth, envelope, health/ready/metrics/pprof, config/env map.
- Protocolos MCP: tipos/messages (tools/resources, sessions, invocations, errors), versionamento e invariantes (tenant_id, scopes, idempotency).
- Runtime: InvokeTool streaming/cancel, rate limit/body cap, retries/backoff+circuit outbound com EnsureTenantHeader, per-tool scopes.
- Tool registry view (usa catálogo do tools-manager) com cache/ETag; sessão plugável.
- Observabilidade/audit: métricas por tool/tenant, tracing, audit redigido.
- Tests: golden MCP msgs, policy matrix, registry cache, retry/circuit, envelope.

## 2) tools-manager (governança/catalogo)
- Scaffold com platform-auth, envelope, migrations/RLS, health/metrics.
- Modelo: tools, tool_versions, tenant_tools (enablement/overrides), policies/quotas; validação JSON Schema com limites.
- Policy/RBAC: allow/deny tenant/agent/scope/env, quotas/rate, precedência determinística.
- Segredos via Vault/KMS (fetch on use, rotacionar, auditar, sem plaintext).
- APIs/gRPC de leitura para gateway/MCP com ETag/If-None-Match; eventos de mudança (Kafka/webhook).
- Tests: migrations/RLS, policy matrix, resolução com overrides, schema limits, secrets guard, ETag/cache, contrato consumer.

## 3) tools-gateway (plano de execução) hardening
- Trocar mock auth por platform-auth; forçar tenant/scope em CRUD/exec; CORS/rate limits básicos.
- Validação/segurança: limites de payload/depth, allowlist de host/protocolo, per-tenant enablement, caps de timeout/top-k.
- Resiliência: retries/backoff + circuit breaker outbound; códigos 4xx/5xx corretos; idempotência/exec ID; DLQ opcional.
- Observabilidade/audit: métricas (latência/erro/retry/rate-limit), tracing inbound/outbound, audit de execuções/OAuth.
- Billing/quotas: emitir eventos de uso (tenant/tool/latência/credits) e aplicar quotas/rate limit.
- OAuth/integrations: provider allowlist, state TTL, token encrypt at rest, máscara de segredos, revogação.
- Tests: auth+tenant isolation, schema + size/host limits, retry/backoff mapping, quotas/billing events, OAuth flows, metrics/tracing emission.

## 4) Adoção de tenant helpers/observabilidade nos serviços existentes
- tenant-manager, billing-service, analytics-query-service, tools-gateway, auth-gateway, voice-gateway, rag-gateway: EnsureTenantHeader outbound, guards em DB/Kafka, envelope helper onde faltar.
- Tests de header/tenant propagation e envelope.

## 5) Integrações de orquestração
- agent-orchestrator: usar cliente MCP e callbacks streaming; feature flag/shadow para comparativo.
- tools-gateway: consumir catálogo do tools-manager com cache/ETag; fallback polling.
- voice/analytics: garantir tenant/session IDs em eventos.

## 6) Observabilidade/SLOs e runbooks
- Dashboards e alertas mínimos (latência/erro per tool/tenant, quota/rate-limit hits, retries/circuit state).
- Runbooks: adicionar/atualizar ferramenta, ativar tenant, rotacionar segredo, lidar com falha downstream/DLQ.

## 7) Release/rollout
- Feature flags para MCP path em agent-orchestrator/tools-gateway.
- Shadow runs → enable por tenant → remover runners legados.

## Checklists
- Config: ISSUER/AUDIENCE/SERVICE_AUDIENCE, REQUIRED_SCOPES, TENANT_CLAIM, JWKS_URL/JWT_SECRET, TLS outbound, POSTGRES_DSN+TLS, CACHE_TTL/ETAG, RATE/QUOTA, RETRY/BACKOFF/CIRCUIT, AUDIT_SINK, TRACE/METRICS exporters, KAFKA/webhook endpoints, Vault/KMS.
- Testes: MCP golden, policy matrix, registry cache/ETag, retries/circuit, quota/rate-limit, envelope, metrics/tracing/audit, secrets guard.
