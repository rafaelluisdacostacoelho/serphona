# platform-mcp Rollout & Feature Flags

## Goals
- Reduce blast radius while adopting the shared MCP stack across services.
- Provide toggles for risk controls (rate limits, audit/tracing sampling, fallback paths).

## Feature Flags / Toggles
- `mcp.enable`: master switch for the platform-mcp invocation stack per service/route.
- `mcp.rate_limit.enabled`: enable/disable rate limiting; per-tenant/tool overrides.
- `mcp.retry.enabled`: enable retries for idempotent tools only.
- `mcp.audit.sampling`: percentage or rules (e.g., per tenant/tool/outcome).
- `mcp.tracing.sampling`: OTEL sampler config per env (e.g., parent-based, ratio).
- `mcp.circuit.enabled`: enable circuit breaker for outbound tool calls.
- `mcp.shadow.enabled`: enable shadow mode (mirror to platform-mcp stack without affecting user).

## Rollout Phases
1) **Shadow mode**
- Keep existing path as source of truth; mirror requests to platform-mcp stack.
- Compare metrics: call counts, errors, latency, rate-limit denials, circuit opens.
- Store audit logs separately; validate tenant/tool labels, request_id/trace_id presence.

2) **Limited enablement**
- Enable `mcp.enable` for a small tenant subset; keep conservative rate limits.
- Gradually raise audit/tracing sampling to target levels.
- Monitor dashboards and alerts (below) continuously.

3) **General availability**
- Enable for all tenants/tools.
- Set final sampling rates; tighten rate limits as needed.
- Remove or narrow shadow mode once parity confirmed.

## Monitoring & Alerts
- Dashboards: invocations (count/latency/error) by tenant/tool; rate-limit denials; circuit breaker opens; audit volume; tracing sampling rates.
- Alerts: sustained error spike per tenant/tool, circuit breaker open rate > threshold, audit sink failures, missing trace/audit for >X% requests.

## Break-glass / Fallback
- Flag to disable retries/circuit for specific tools if they are non-idempotent or unstable.
- Ability to bypass platform-mcp stack entirely (per route/tenant) if blocking issues arise.
- Keep legacy path available during rollout; document reversion steps.

## Checklist (per environment)
- [ ] Flags wired and configurable at runtime (env vars or config service).
- [ ] Shadow mode path validated and not user-visible.
- [ ] Dashboards and alerts deployed (metrics/tracing/audit).
- [ ] Rate limits per tenant/tool reviewed and set conservatively.
- [ ] Retry/circuit configs reviewed for idempotency.
- [ ] Audit/tracing sampling set; sinks reachable and tested.
- [ ] Response envelope conformance verified in canaries (trace_id/request_id present).
- [ ] Run `go test ./...` and `gosec ./...` before promotion.

## References
- Config matrix: docs/architecture/CONFIG-MATRIX-platform-mcp-en-US.md
- Response envelope: docs/architecture/RESPONSE-ENVELOPE-CONTRACT-en-US.md
- RLS/tenant: docs/architecture/TENANT-RLS-GUIDANCE-en-US.md
