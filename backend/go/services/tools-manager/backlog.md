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
1) **Scaffolding and config**
- Create service skeleton (cmd/server) with platform-auth middleware/interceptors, response envelope, health/ready/metrics/pprof, config loader/env map.
- Add migrations runner (Postgres with RLS), Makefile/Dockerfile, CI wiring.

2) **Catalog data model**
- Tables: tools, tool_versions, tenant_tools (enablement/overrides), categories/tags; include tenant_id, version status (draft/published/deprecated), owners.
- Constraints: uniqueness by name+version, per-tenant enablement unique, indexes for tenant/name/category.

3) **Validation and safety**
- JSON Schema validation with depth/field/size caps for input/output; host/protocol allowlist metadata; limits on timeouts, retries, payload sizes.
- Rejection of invalid or unsafe definitions at write time; DTOs with explicit limits.

4) **Policy/RBAC and quotas**
- Allow/deny evaluator: tenant, agent, scopes/roles, environment, quotas/rate limits; deterministic precedence.
- Admin APIs to set policies and quotas; per-tenant enable/disable; audit each decision/change.

5) **Secrets and credentials**
- Integrate Vault/KMS for provider secrets per tenant/tool; fetch-on-use with caching TTL; mask in logs; rotation API/hooks.
- Audit: who accessed/rotated; deny cross-tenant access.

6) **Integrations and sync**
- Read APIs/gRPC to deliver resolved tool definitions (with overrides) by tenant for Tools Gateway and platform-mcp; pagination + ETag/If-None-Match.
- Change events to Kafka/webhook for cache invalidation in gateway; fallback polling endpoint; contract tests for consumers.
- MCP catalog view (schema compatible with platform-mcp/tool registry) for agent-orchestrator discovery.

7) **Observability and audit**
- Metrics: writes/reads, cache hits, policy decisions, quota hits, secret fetches, errors; tracing with tenant/user/tool/version.
- Audit log: create/update/publish/deprecate actions with diff hash and actor; sampling controls.

8) **Testing and quality**
- Migrations/RLS tests; policy matrix; catalog resolution with overrides; schema limit tests; secret access tests; cache/ETag tests; contract tests for gateway consumer.
- Static analysis (gosec), fuzz of input DTOs, load smoke for read APIs.

9) **Docs and runbooks**
- How-to: add/update/publish tool, set policies/quotas, enable tenant, rotate secrets, rollback.
- Versioning/breaking-change checklist; MCP/gateway consumer integration notes; ops dashboards/alerts and oncall runbook.

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
