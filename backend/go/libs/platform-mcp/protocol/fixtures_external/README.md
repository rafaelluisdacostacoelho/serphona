# MCP External Fixtures

Purpose: versioned, public JSON fixtures for MCP requests/events, aligned to `PROMPTS-YAML-SPEC` (tools of type `mcp`) and the response envelope contract.

- Format: JSON (canonical) with `version` field; current: `v1`.
- Location: `protocol/fixtures_external/<version>/...`.
- Scope: invocation requests and streaming events (result/error/cancel) shaped per `protocol.InvocationRequest` and `protocol.InvocationEvent`.
- Envelope: result events carry the standard response envelope (`data`, `meta.trace_id`, `meta.request_id`).
- Compatibility: `v1` maps to `protocol.CurrentVersion` and `response` helpers; future versions must remain side-by-side.

See `v1` files for concrete shapes.
