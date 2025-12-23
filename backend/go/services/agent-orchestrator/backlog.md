# Agent Orchestrator — Backlog (en-US)

## Status snapshot
- Orchestrates tools/LLM calls; should enforce tenant_id and guardrails. Current implementation not reviewed here—needs a quick audit.
- MCP integration planned to resolve tools via Tools Gateway; ensure fallback to HTTP when MCP unavailable.

## Open items (to validate/implement)
1) **Auth & tenant isolation**: require JWT with tenant_id; propagate to downstream tools/LLM; add middleware tests.
2) **Prompt/guardrails**: enforce citation/ACL checks before context assembly; rate limits per channel (voice/text) and per tenant.
3) **Tool resolution**: integrate MCP client; caching of tool catalog; health checks and fallback path.
4) **Observability**: tracing (per turn), structured logs with tool/tenant, metrics for latency, error rate, tool success/fail, and LLM token usage.
5) **Billing hooks**: emit usage events (prompt/completion tokens, tool calls) keyed by tenant and channel.
6) **Resilience**: retries/backoff for tools; circuit breakers and DLQ for failed tool invocations; timeouts per step.
7) **Conversation state**: persistence strategy (Redis/DB) with TTL; ensure RLS/namespace partitioning.
8) **Testing**: contract tests for prompt flows; golden tests for system prompts; integration tests with tool fakes; load tests for concurrency.
9) **Runbooks**: tool/LLM outage handling, DLQ replay, catalog refresh failures, rate-limit escalations.

## Config to surface (verify in code)
- Auth issuer/audience/scopes; tenant_id claim mapping.
- LLM provider/model, temperature/top_p, max tokens, timeouts.
- Tool resolver endpoints (MCP/HTTP), cache TTL, retry/backoff.
- Rate limits per tenant/channel; guardrail toggles; logging/trace exporters.
- Billing/usage topic or sink.

## Test coverage checklist
- [ ] Authz middleware
- [ ] Prompt/guardrail enforcement
- [ ] Tool resolution (MCP + fallback)
- [ ] Retry/circuit on tool failures
- [ ] Observability (metrics/traces present)
- [ ] Billing events emitted

## Next suggested steps
- Do a code/config pass to replace placeholders with concrete details and mark what’s already done.
- Add focused tests for auth + tool resolution + guardrails; wire metrics/traces if missing.
