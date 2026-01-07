# Tools Manager — Backlog (governance plane, MVP Serphona)

## Objective
Provide the governed tool catalog and policy control-plane for Serphona. Tools Manager owns tool definitions, versions, per-tenant enablement, policies/quotas, and secrets, and publishes resolved configurations to execution planes (Tools Gateway, platform-mcp) and agent-orchestrator.

## Definition of done (MVP)
- Service scaffold with platform-auth (JWT/scopes, tenant guard), shared response envelope, health/ready/metrics/pprof, migrations wired, CI task.
- Multi-tenant catalog: tool definitions + versions, tenant overrides/allow-list, ownership metadata, tags/categories, deprecation flags.
- Policy/RBAC and quotas: allow/deny per tenant/tool/agent, scopes/roles, rate/usage quotas, enable/disable per tenant; deterministic precedence.
- Secrets/credentials: per-tenant/provider secrets stored via Vault/KMS (no plaintext at rest); rotation hooks and audit log.
- Validation and safety: JSON Schema validation (depth/size caps) for input/output; outbound host/protocol allowlist metadata; idempotency/versioning rules for definitions.
- Integrations: API/gRPC to fetch resolved tool definitions by tenant; change events to Tools Gateway via Kafka/webhook; MCP-ready catalog exposure for platform-mcp/agent-orchestrator.
- Observability/audit: metrics (catalog reads/writes, policy decisions), tracing, structured audit for changes (who/what/tenant/version), sampling knobs.
- Tests: migrations + RLS, policy matrix, catalog resolution with overrides, schema validation limits, secret access guards, integration contract for gateway sync.
- Docs/runbooks: how to add/update/publish a tool, rotate secrets, enable tenant, rollback, and versioning/breaking-change checklist (en/pt-BR).

## Backlog (ordered)

- [ ] Scaffolding and config
	- [ ] Skeleton em `cmd/server` com platform-auth (middleware/interceptors), envelope de resposta, health/ready/metrics/pprof, carregador de config/env map
	- [ ] Runner de migracoes (Postgres com RLS), Makefile/Dockerfile, CI

- [ ] Catalog data model
	- [ ] Tabelas: tools, tool_versions, tenant_tools (enablement/overrides), categories/tags; campos tenant_id, status (draft/published/deprecated), owners
	- [ ] Constraints: unicidade name+version; enablement por tenant unico; indexes por tenant/name/category

- [ ] Validation and safety
	- [ ] Validacao JSON Schema com limites de profundidade/campos/tamanho para input/output
	- [ ] Metadata de allowlist host/protocol e limites de timeout/retries/payload
	- [ ] Rejeicao de definicoes invalidas ou inseguras no write; DTOs com limites explicitos

- [ ] Policy/RBAC and quotas
	- [ ] Avaliador allow/deny por tenant/agent/scopes/roles/ambiente com quotas/rate e precedencia deterministica
	- [ ] APIs admin para politicas/quotas, enable/disable por tenant; auditar cada decisao/alteracao

- [ ] Secrets and credentials
	- [ ] Integrar Vault/KMS para segredos por tenant/tool; fetch-on-use com cache TTL; mascarar logs; rotacao API/hooks
	- [ ] Auditar quem acessou/rotacionou; negar acesso cross-tenant

- [ ] Integrations and sync
	- [ ] APIs/gRPC de leitura para definicoes resolvidas por tenant (Tools Gateway, platform-mcp) com paginacao e ETag/If-None-Match
	- [ ] Eventos de mudanca via Kafka/webhook para invalidação de cache; endpoint de polling fallback; testes de contrato dos consumidores
	- [ ] Vista MCP (schema compatível com platform-mcp/tool registry) para agent-orchestrator

- [ ] Observability and audit
	- [ ] Metricas: writes/reads, cache hits, policy decisions, quota hits, secret fetches, errors; tracing com tenant/user/tool/version
	- [ ] Audit log de create/update/publish/deprecate com diff hash e actor; sampling controls

- [ ] Testing and quality
	- [ ] Testes de migracoes/RLS; matriz de politica; resolucao com overrides; limites de schema; acesso a segredos; cache/ETag; contratos com gateway consumer
	- [ ] Static analysis (gosec), fuzz de DTOs de input, load smoke para read APIs

- [ ] Docs and runbooks
	- [ ] How-to: add/update/publish tool, politicas/quotas, habilitar tenant, rotacionar segredos, rollback
	- [ ] Checklist de versionamento/breaking-change; notas de integracao MCP/gateway; dashboards/alertas e runbook de oncall

## Config to surface
- ISSUER/AUDIENCE/REQUIRED_SCOPES, TENANT_CLAIM, JWKS_URL/JWT_SECRET; POSTGRES_DSN + TLS/pool/migrations toggle; VAULT/KMS settings; CACHE_TTL/ETAG; RATE/QUOTA defaults; KAFKA/Webhook endpoints; TRACE/METRICS exporters; LOG level/format.

## Test coverage checklist
- [ ] RLS/tenant isolation and migrations
- [ ] Schema validation limits (depth/size/fields) and host/protocol allowlist
- [ ] Policy allow/deny matrix and quotas
- [ ] Catalog resolution with tenant overrides and ETag/cache
- [ ] Secret access/rotation guards
- [ ] Gateway/platform-mcp consumer contract
- [ ] Metrics/tracing/audit emission

## Next steps
- Ship scaffold + data model + validation; implement policy engine and secrets; expose read API + change events for gateway/mcp; add observability and contract tests; document workflows.

## Referências
- Este serviço depende do [BACKLOG-AUTH.md](../../../BACKLOG-AUTH.md) para alinhamento com autenticação e autorização multi-tenant.
