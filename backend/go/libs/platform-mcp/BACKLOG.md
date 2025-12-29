# platform-mcp — Backlog (Library, MVP Serphona)

## Objective
Provide an MCP library that Serphona services (platform-mcp service, agent-orchestrator, tools-gateway) can embed to handle MCP protocol, policy, registry, invocation, and observability with strict multi-tenancy and platform-auth alignment.

## Definition of done
- Protocol/types defined for MCP messages (tools/resources, sessions, invocations, errors) with versioning and invariants: tenant_id required, scopes/roles hints, idempotency keys, error codes.
- Policy/RBAC engine: allow/deny by tenant, agent, scopes/roles, environment and rate limits; deterministic precedence; table-driven tests.
- Tool registry multi-tenant: file + Postgres loader with per-tenant allow-list; cache with TTL/ETag; ListTools/DescribeTool APIs aligned to PROMPTS-YAML-SPEC.
- Invocation runtime: streaming progress + cancel, rate limiting and body-size caps, retries/backoff + circuit breaker for outbound calls, EnsureTenantHeader/service identity on outbound, response envelope helpers.
- Session manager (pluggable store) with tenant isolation and expiry, or clear hooks if hosting service keeps session state.
- Observability/audit: metrics (calls/latency/errors by tenant/tool), tracing with request/trace/tenant/session IDs, structured/redacted audit logs; sampling hooks.
- Test suite: golden MCP message vectors (ok/error/cancel), policy matrix, registry cache/reload, retry/circuit behavior, envelope contract; fakes for platform-auth/JWKS.
- Docs: how to embed/configure, add a tool with scopes, auth/tenant wiring, breaking-change checklist, config matrix; examples in en-US/pt-BR.

## Cards (prioridade e checkpoints)

**Card 1 — Protocol & Contracts**
- [x] Definir invariantes de protocolo (tenant_id obrigatório, validação de tool/invocation, códigos de erro incluindo cancelled).
- [x] Vetores básicos MCP (ok/erro/cancel/unsupported) em testes de fixtures internos.
- [x] Formalizar esquema/compatibilidade (JSON canon, v1), publicar fixtures externos versionados.

**Card 2 — Policy / RBAC**
- [x] Avaliador allow/deny com precedence e rate-limit básico + testes.
- [x] Métricas para decisões de política.

**Card 3 — Registry multi-tenant**
- [x] Loader file + Postgres por tenant.
- [x] Cache com TTL/ETag e testes de not-modified/refresh.
- [x] Alinhar payloads com PROMPTS-YAML-SPEC e definir allow-list por tenant.

**Card 4 — Invocation runtime & helpers**
- [x] Guard de payload/timeout.
- [x] Resilient executor (retry/backoff + circuit breaker).
- [x] Helper EnsureTenantHeaders.
- [x] Executor cancelável.
- [x] Streaming de progresso/cancel no executor real e propagação de cancel downstream.
- [x] Enforce de max output bytes.
- [x] Rate limit por tool/tenant.
- [x] Envelope helpers alinhados ao RESPONSE-ENVELOPE.

**Card 5 — Session management**
- [x] Store in-memory.
- [x] Store Postgres com testes.
- [x] Propagar session_id/tenant em audit/telemetria.

**Card 6 — Observabilidade e audit**
- [x] Métricas básicas (contagem/latência/outcome) com enrichers (cache_hit/policy_decision) e sink Prometheus.
- [x] Audit observer com sink plugável.
- [x] Tracing: spans para sessão e invocação com tenant/tool/request/trace IDs.
- [x] Audit: sampling/roteamento e integração OTEL.

**Card 7 — Testes e qualidade**
- [x] Policy matrix completa e precedence.
- [x] Retry/backoff/circuit e rate-limit com fakes.
- [x] Envelope contract + fuzz de headers.
- [x] Static analysis (gosec) e license checks.

**Card 8 — Docs & rollout**
- [x] How-to embed/config (platform-mcp, agent-orchestrator, tools-gateway) en-US/pt-BR.
- [x] Wiring/fixtures quickstart en-US/pt-BR (docs/INTEGRATION-*).
- [x] Config matrix (auth, cache, retry/backoff/circuit, metrics/audit, RLS/tenant wiring).
- [x] Rollout: feature flag/shadow mode guidance; breaking-change checklist.

## Config to surface (docs/examples)
- ISSUER/AUDIENCE/SERVICE_AUDIENCE, REQUIRED_SCOPES, TENANT_CLAIM, JWKS_URL/JWT_SECRET, TLS_* for outbound, POSTGRES_DSN, CACHE_TTL/ETAG, RATE_LIMITS, RETRY/BACKOFF/CIRCUIT settings, AUDIT_SINK, TRACE/METRICS exporters.

## Test coverage checklist
- [x] MCP message golden vectors (ok/error/cancel/unsupported)
- [x] Policy allow/deny matrix and precedence
- [x] Registry cache/fallback/ETag behavior
- [x] Invocation retries/backoff/circuit and rate limits
- [x] Envelope contract and header parsing fuzz
- [x] Metrics/tracing/audit emission

## Next steps
- Lock protocol/schema and policy precedence; implement registry + invocation runtime with tests; add observability/audit and docs; wire examples for agent-orchestrator/tools-gateway.
