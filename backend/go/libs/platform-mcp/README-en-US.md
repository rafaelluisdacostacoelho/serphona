# platform-mcp (scaffold)

Purpose: shared MCP building blocks (schemas, policy checks, discovery helpers, audit logging) for MCP servers and clients in Serphona.

Status: scaffold only — no code yet.

Suggested contents (next steps):
- `resource/`: types for MCP resources/tools with input/output schemas and idempotency hints.
- `policy/`: evaluation helpers for tenant/agent/environment allow/deny, rate limits, scopes.
- `client/`: thin client to resolve resources and execute calls with tracing/metrics.
- `audit/`: logging helpers (hash input/output, status, latency, tenant/agent IDs).
- `test/`: contract tests/mocks for policy and client behavior.
