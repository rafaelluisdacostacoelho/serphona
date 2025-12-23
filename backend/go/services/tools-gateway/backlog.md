# Tools Gateway — Backlog (en-US)

## Status snapshot
- Acts as façade for tool invocation and routing; integrates with platform-auth for tenant-scoped access.
- MCP alignment pending; current state assumed to expose HTTP/gRPC interfaces (verify in code/config).
- No recent audit captured here; update after reviewing service code and configs.

## Open items (fill with specifics after code review)
1) **Auth & ACL**: confirm middleware enforcing tenant_id/scopes on all endpoints; add contract tests.
2) **Schema validation**: ensure request/response schemas for registered tools; add JSON schema validation and error mapping.
3) **Rate limits/quotas**: per-tenant and per-tool limits; surface metrics for throttling and 429s.
4) **Observability**: tracing with tenant_id/tool, structured logs; Prometheus metrics for latency, errors, and downstream failures.
5) **MCP integration**: expose catalog/discovery for Agent Orchestrator; map tool metadata and auth to MCP.
6) **Billing hooks**: emit usage events per tool call (success/error, token usage if applicable).
7) **Resilience**: retries/backoff for downstream calls; circuit breakers and DLQ for failed tool executions.
8) **Testing**: contract/golden tests for tool schemas; integration tests with fakes for downstream services; load tests for concurrency.
9) **Runbooks**: tool onboarding, rollback/disable flow, DLQ drain and replay, incident steps for quota/rate-limit misfires.

## Config to surface (verify in service code)
- Auth settings (issuer/audience, JWKS, required scopes), tenant_id propagation.
- Server listen address/ports; timeouts; max payload size.
- Downstream endpoints per tool; retry/backoff/circuit settings.
- Metrics/tracing exporters; log level/format.
- Billing/usage event topic or sink if present.

## Test coverage checklist
- [ ] Authz happy-path and rejection
- [ ] Schema validation per tool
- [ ] Rate limit/quota enforcement
- [ ] Downstream failure → retry/circuit/DLQ
- [ ] Observability (metrics/traces present)
- [ ] Integration with Agent Orchestrator/MCP client

## Next suggested steps
- Do a quick code/config audit to replace placeholders above with concrete endpoints/configs and mark completed items.
- Add targeted tests for auth + schema validation; wire metrics/traces if missing.
