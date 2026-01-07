# Tools Gateway — Backlog (en-US)

## Status snapshot
- Audit complete: Gin HTTP only; mock auth middleware injects random tenant/user UUIDs (no JWT/scopes). Tool CRUD, integration CRUD, OAuth flows, and execution path exist; tenant scoping is unchecked and configs lack validation. No rate limits, quotas, billing hooks, or observability beyond defaults. DB connections use env DATABASE_URL without migrations or TLS. Execution uses HTTP client with schema validation but no retries/backoff; credits consumed stored but not billed.

## Audit findings
- Auth/tenant: no real auth middleware; tenant_id/user_id are random per request; no scope/role checks; integrations and tools not filtered by authenticated tenant except by context when present; CORS/rate limits absent.
- Validation/schema: input/output schemas checked via validator, but schema validity only shallow (IsValidSchema) and no size/field limits; ExecuteTool trusts tenant context and allows any tool ID across tenants unless tenant-tool mapping is configured; lacks payload size caps.
- Execution/resilience: HTTP client executes downstream without retries/backoff/circuit breaker; timeout fixed at 30s; errors return 200 with error body; no DLQ/idempotency; no per-tool concurrency caps.
- Data layer: GORM with DATABASE_URL default; migrations not invoked; tables and uniqueness constraints assumed; no RLS/tenant enforcement in queries (tenant filter optional). No TLS/pool tuning.
- Observability: no metrics, tracing, or structured logging in handlers/usecases; health only. No audit logs for tool executions or OAuth.
- Billing/quotas: credits consumed recorded but no billing event or quota enforcement per tenant/tool/user; no rate limits.
- OAuth/integrations: OAuth service supports state/PKCE/token refresh but no persistence validation or replay protection checks; provider allowlist absent; token revocation best-effort; tokens stored without encryption at rest.
- Security: Mock auth leaves endpoints open; no input size limits; downstream URLs from tool definitions not validated/allowlisted; secrets (client secret, API keys) not masked; HTTPS/TLS not exposed.
- MCP/Agent alignment: no MCP interface/catalog; execution path is HTTP-only; no gRPC.

## Action items

- [ ] Auth & tenant
	- [ ] Substituir mock auth por JWT (issuer/audience/scopes) e tenant guard em tool/integration/execute
	- [ ] Enforce role-based access e CORS/rate limits
	- [ ] Remover IDs aleatorios; sempre filtrar por tenant autenticado

- [ ] Validation & safety
	- [ ] Limitar payload (size) e profundidade/campos de schema
	- [ ] Allowlist de host/protocolo para execucoes downstream; checar enablement por tenant/tool
	- [ ] Cap de TopK/limits e validacao forte de schemas

- [ ] Resilience
	- [ ] Retries/backoff e circuit breaker por tool; timeouts configuraveis
	- [ ] Idempotencia de execucao e DLQ/erro queue para falhas
	- [ ] Mapear erros para 4xx/5xx (nao 200) e incluir trace_id

- [ ] Data/migrations
	- [ ] Rodar migracoes no startup; constraints de unicidade (tool name, tenant-tool) e indices
	- [ ] RLS/tenant scoping nas queries e filtros obrigatorios
	- [ ] Configurar TLS/pool do DB

- [ ] Observability/audit
	- [ ] Metricas (latencia, erros, rate-limit blocks, retries) e tracing inbound/outbound
	- [ ] Audit estruturado para execucoes e eventos OAuth

- [ ] Billing/quotas
	- [ ] Quotas/rate limits por tenant/tool; bloqueio e metricas
	- [ ] Eventos de uso (tool, latencia, status, credits) para billing/analytics; modelo de preco

- [ ] OAuth/integrations
	- [ ] Provider allowlist; state storage seguro + expiracao; PKCE
	- [ ] Token encryption at rest e revogacao; isolamento por tenant
	- [ ] Mascara de segredos em logs

- [ ] MCP/Agent
	- [ ] Expor catalog/discovery endpoint ou gRPC para agent-orchestrator/MCP
	- [ ] Padronizar schema/versao das ferramentas

- [ ] Testing
	- [ ] Contratos de schema, isolamento auth/tenant, caminhos de erro de execucao
	- [ ] OAuth com fake provider; retries/backoff; metrics/tracing emission

- [ ] Runbooks
	- [ ] Onboarding/desabilitar tool, rotacionar segredos, lidar com outages/DLQ replay
	- [ ] Ajuste de quota/rate-limit e falhas de refresh de token

- [ ] Response contract
	- [ ] Adotar envelope comum (success meta; error code/message/details/trace_id) em todos os handlers
	- [ ] Ajustar mapeamento de erros de execucao e testes

## Config to surface
- HTTP addr, timeouts, body size, CORS, rate limits.
- Auth issuer/audience/JWKS, required scopes/roles; tenant claim name.
- DATABASE_URL with TLS/pool/migrations toggle.
- Downstream execution settings per tool: allowed hosts, timeouts, retries/backoff, circuit breaker, max concurrency.
- OAuth: providers allowlist, client IDs/secrets, state TTL storage, token encryption, redirect URLs.
- Observability exporters (Prometheus/Otel), log level/format; billing/usage event sink.

## Test coverage checklist
- [ ] JWT auth + tenant isolation on tool/integration CRUD and execute
- [ ] Schema validation and payload size/host allowlist enforcement
- [ ] Execution retries/backoff and error/status mapping (non-200s)
- [ ] Quotas/rate limits and billing usage events
- [ ] OAuth flows (state TTL/PKCE, token refresh/revoke)
- [ ] Metrics/tracing/audit emitted and migrations applied

## Next suggested steps
- Replace mock auth with platform-auth middleware and tenant scoping; add validated config (limits, allowlists) and migrations. Then implement retries/backoff, metrics/tracing, quotas/billing events, and secure OAuth/integration handling before MCP integration.

## Referências
- Este serviço depende do [BACKLOG-AUTH.md](../../../BACKLOG-AUTH.md) para alinhamento com autenticação e autorização multi-tenant.
