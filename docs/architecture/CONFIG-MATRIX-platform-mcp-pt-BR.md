# Matriz de Configuração do platform-mcp (platform-mcp, agent-orchestrator, tools-gateway)

> Objetivo: alinhar configs por domínio (auth, registry, invocação, observabilidade, rollout) e manter paridade entre ambientes.

## Auth / Identidade
- ISSUER / AUDIENCE / SERVICE_AUDIENCE: devem casar com o issuer do platform-auth e audiences dos serviços.
- TENANT_CLAIM: claim com tenant_id (obrigatório para RLS e métricas).
- REQUIRED_SCOPES: por ferramenta/rota; sincronizar com regras de política.
- JWKS_URL ou JWT_SECRET: por ambiente; preferir JWKS com cache; rotacionar segredos.
- HEADERS: propagar `x-request-id`, `traceparent`, `authorization` (apenas inbound). Não propagar cookies.

## Registry
- POSTGRES_DSN: RLS habilitado; políticas por tenant aplicadas (ver TENANT-RLS-GUIDANCE).
- CACHE_TTL / ETAG_FN: alinhar com frequência de mudanças; TTL menor em dev.
- ALLOW_LIST_SOURCE: allow-list por tenant (tabela DB ou arquivo).
- FALLBACK_LOADER: loader de arquivo para cold-start/dev; garantir filtro por tenant.
- MCP_TOOLS_ROOT: obrigatório ao habilitar fallback de arquivo; isola leituras em um diretório raiz para evitar path traversal.

## Policy / RBAC
- EVALUATOR: memória para testes/dev; banco em produção.
- RATE_LIMIT_POLICY: regras por tenant/tool; default deny se não houver política.
- MÉTRICAS: habilitar `mcp_policy_*` com `MetricsEvaluator`; rótulos env/tenant/tool/rule.

## Runtime de Invocação
- MAX_BODY_BYTES / MAX_OUTPUT_BYTES: limitar entradas/saídas conforme ferramentas.
- TIMEOUT: padrão por tool; circuit breaker por dependência.
- RETRY/BACKOFF: apenas para ferramentas idempotentes; backoff exponencial (50ms<<tentativa).
- CIRCUIT_BREAKER: threshold/reset por tool; comece conservador.
- RATE_LIMITS: bucket por tenant/tool; ajustar burst para UX.
- ENABLE_CANCEL: true para streaming; garantir que downstream honre context.

## Headers / Outbound
- `EnsureTenantHeaders` deve adicionar `x-tenant-id`, `x-request-id`, `traceparent`, `x-service-id` nas chamadas a tools.
- Manter headers em minúsculas nos gateways.
- Remover headers sensíveis ao cruzar domínios de confiança.

## Observabilidade / Audit
- MÉTRICAS: sink Prometheus; expor endpoints para scrape.
- TRACING: exportador OTEL (OTLP/gRPC); tracer `platform-mcp`; sampling por ambiente.
- AUDIT: sink (arquivo/OTEL); habilitar sampling/roteamento; redigir PII; incluir tenant/tool/request/session.
- LABEL ENRICHERS: cache_hit, policy_decision.

## Envelope de Resposta
- Use `response.Success` / `response.Error`; inclua request_id e trace_id.
- HTTP: seguir RESPONSE-ENVELOPE-CONTRACT; gRPC: metadata traceparent/x-request-id.
- Fuzz: testes de parsing de headers x-request-id/traceparent para evitar injeção.

## Rollout / Ops
- Feature flags: rate-limit por tool, sampling de audit/tracing.
- Shadow mode: espelhar chamadas via stack platform-mcp e comparar métricas/audit antes do corte.
- Dashboards: chamadas/latência/erro por tenant/tool; negações de rate limit; circuit breaker aberto; volume de audit.
- Alertas: circuit open, 4xx/5xx sustentado por tenant/tool, falhas no sink de audit.

## Paridade por Ambiente
- ✅ Dev: policy em memória, registry de arquivo, OTEL local, rate limit permissivo.
- ✅ Staging: registry Postgres com RLS, policy em DB, OTEL de staging, audit com sampling.
- ✅ Prod: RLS estrita, policy em DB obrigatório, rate/circuit conservadores, audit em sink durável, sampling ajustado.
