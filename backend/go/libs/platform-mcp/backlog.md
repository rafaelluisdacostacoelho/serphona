# platform-mcp — Backlog (en-US)

## Status snapshot
- Scaffold only (no code); intended to provide MCP resource types, policy checks, client helpers, and audit/logging.

## Gaps and tasks (ordered)
1) **Resource/schema model**
   - Define types for resources/tools with input/output schemas, idempotency keys, and error codes.
   - Include tenant_id, scopes, and ACL hints for each resource; document invariants.
2) **Policy engine**
   - Allow/deny evaluation based on tenant, agent, environment, rate limits; support reusable policies.
   - Add tests for policy combinations and precedence.
3) **Client helpers**
   - Resolve resources from catalog (Tools Gateway) with caching; execute calls with tracing/metrics.
   - Retries/backoff and circuit breakers for tool execution; DLQ or audit on failure.
4) **Audit/observability**
   - Structured audit logs with hashed inputs/outputs, status, latency, tenant/agent IDs.
   - Prometheus metrics (calls, latency, errors) and tracing propagation.
5) **Docs**
   - Usage examples for server/client side, catalog resolution flow, and policy configuration.
6) **Testing**
   - Contract tests for schema validation; policy unit tests; client retry/circuit tests with fakes.

## Config to surface (doc)
- Catalog endpoint, cache TTL; tracing/metrics exporters; retry/backoff/circuit settings; audit sink.

## Test coverage checklist
- [ ] Resource/schema validation
- [ ] Policy allow/deny matrix
- [ ] Client retry/circuit behavior
- [ ] Audit/metrics emitted
- [ ] Catalog cache/fallback behavior

## Next steps
- Design minimal types and policy helpers, then add client/audit scaffolding with tests; update README to move beyond “scaffold”.
