# Agent Orchestrator — Backlog (en-US)

## Status snapshot
- Audit complete: service starts Gin HTTP without auth/tenant enforcement, uses Redis for sessions and optional Postgres agents, no metrics/tracing/logging beyond prints, no rate limiting/guardrails, Tools Gateway client is optional but not exercised, tool calls stubbed, and LLM client pool is a placeholder. No config validation or env defaults for safety. Sessions/agents are multi-tenant but tenant comes only from request body/query (not auth) and is not propagated to LLM/tool calls.

## Audit findings
- Boot/config: env parsing is ad-hoc; no required checks; server enables CORS *; no TLS; request IDs are random without trace correlation; no shutdown hooks for background workers.
- Auth/tenant: no JWT/auth middleware; tenant/user IDs taken from payload/query; no RLS guard; no propagation to LLM/tool calls.
- API handlers: lack validation, pagination, and rate limiting; agent/session CRUD exposed without auth; errors leak internals; no idempotency for messages.
- Session management: Redis repo default TTL 24h; no per-tenant quotas; expired session handling only on read; messages returned without redaction.
- Message processing: tool integration TODO (convertTools empty, processToolCalls may be incomplete); default agent selection is first active; no guardrails, safety filters, or prompt templating; no streaming; no timeouts for LLM/tool calls; no retries/backoff/circuit except unused breaker package; token/cost calc exists but not billed/emitted.
- LLM/tool clients: clientPool.SetupDefaultClients uses OpenAI key only; Anthropic unused; Tools Gateway client optional and not authenticated; no caching of tool schemas; no schema validation; no fallback to MCP; no DLQ for tool failures.
- Observability: metrics package logs instead of Prometheus; no tracing/logging integration; no request/tenant IDs in logs; no health/ready metrics.
- Resilience: circuit breaker exists but unused; no timeouts per external call; no concurrency limits; no dead-letter for failed tool chains.
- Data model: agents/sessions have tenant_id but no enforcement in repos; Postgres repo not inspected for RLS; Redis repo likely stores raw JSON without encryption.
- Docs/runbooks: README missing; no runbooks for outages/DLQ; no config matrix.

## Action items
1) Auth/RLS: add platform-auth middleware (JWT with tenant/user/scopes), enforce tenant on all queries (Redis keys + Postgres RLS), propagate tenant/user to LLM/tool headers and traces.
2) Config: centralize config struct with validation (HTTP addr, Redis, DB, Tools Gateway/MCP endpoints, LLM keys/models, timeouts, rate limits), env prefix, and required checks; add sane defaults and TLS options.
3) Guardrails/rate limits: add per-tenant and per-channel rate limiting, message size limits, PII redaction, allowed tools list, and safety filters before LLM/tool calls; add prompt templates with citations/ACL checks.
4) Tooling: integrate Tools Gateway + MCP client with schema cache, retries/backoff/circuit, per-tool timeouts, DLQ for failed tool calls, and fallback to HTTP when MCP unavailable.
5) LLM pipeline: support multiple providers (OpenAI/Anthropic), timeouts, retries, streaming, function/tool call execution flow, token/cost tracking, and configurable system prompts; add delegation chain limits.
6) State: session TTL configurable, cleanup worker, pagination for messages, idempotency keys for message posts, and optional persistence of agents with migrations + RLS; encrypt sensitive session data if needed.
7) Observability: add tracing (HTTP, LLM, tools), structured logging with request/trace/tenant IDs, Prometheus metrics (HTTP/gRPC latency, LLM/tool latency/success, sessions, tokens), and health/ready endpoints with dependencies.
8) Resilience: wire circuit breaker/timeouts for LLM/tool/DB/Redis; add retries with backoff + jitter; configure DLQ topic for failed tool executions and optionally failed LLM calls; add bulkhead/concurrency limits.
9) Billing/events: emit usage events (tokens, tool calls, duration) with tenant/channel metadata to billing/events topics; include error reasons.
10) API hygiene: add validation, consistent error codes, pagination, sorting, filtering; protect agent/session routes with auth + scope checks; add OpenAPI spec and contract tests.
11) Docs/runbooks: add README/PLANNING alignment, config matrix, SLOs, runbooks for LLM/tool outages, DLQ replay, and rate-limit escalation; example deployments (env, compose, k8s).
12) Testing: unit tests for handlers (auth/tenant), session/agent services, LLM/tool orchestration with fakes, rate limits, retries/circuit, metrics/traces; integration tests with fake Tools Gateway + fake LLM; load tests for concurrency.

## Config to surface
- HTTP_ADDR, TLS cert/key (if enabled); REDIS_ADDR/PASSWORD/DB/TTL; DATABASE_URL + RLS; TOOLS_GATEWAY_URL / MCP_ENDPOINT + auth; OPENAI_API_KEY/ANTHROPIC_API_KEY, MODEL defaults, LLM_TIMEOUT/RETRIES; RATE_LIMITS per tenant/channel; DLQ topic/endpoint; METRICS_PORT/OTEL_EXPORTER; AUTH_ISSUER/AUDIENCE/SCOPES/TENANT_CLAIM.

## Test coverage checklist
- [ ] AuthZ (JWT/scopes/tenant) on all routes
- [ ] Tool resolution (Tools Gateway + MCP) with retries/backoff/circuit and fallback
- [ ] LLM call flow with tool calls, timeouts, streaming path
- [ ] Guardrails/rate limits and payload validation
- [ ] Observability: traces + Prometheus metrics emitted
- [ ] Billing/usage events on tokens/tool calls
- [ ] DLQ handling for failed tool executions

## Next steps
- Add config/auth middleware and tenant enforcement first; then wire observability + rate limits; implement tool/LLM orchestration with retries/timeout/circuit and billing events; finish with docs/runbooks and integration/load tests.
