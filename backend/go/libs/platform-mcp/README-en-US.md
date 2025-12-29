# platform-mcp

Shared building blocks for Model Context Protocol (MCP) servers and clients across Serphona. These packages provide common schemas, tool discovery, policy enforcement, invocation orchestration, and envelopes for consistent responses.

## Packages
- errors/: common error codes for invocation and platform errors.
- protocol/: shared request/response/tool schemas and validation.
- loader/: safe tool-manifest loader (JSON), enforces MCP_TOOLS_ROOT, blocks traversal, validates tools.
- registry/: registries (memory, cached, allow-list, Postgres) with ETag-aware list/describe helpers.
- invoke/: executors (static map, cancel-aware, observed for metrics/audit, resilient with backoff/circuit breaker) and progress/audit helpers.
- policy/: in-memory rule engine with tenant/tool/env matching, scope requirements, and per-rule rate limiting.
- response/: success/error envelopes with trace and request IDs plus pagination.
- session/: in-memory and Postgres-backed session stores.

## Usage overview
- Tool manifests: provide JSON files; set MCP_TOOLS_ROOT to confine lookups. Traversal and non-regular files are rejected.
- Registries: use ListToolsWithETag/DescribeToolWithETag to honor conditional requests. CachedRegistry adds TTL-based refresh; AllowListRegistry filters per-tenant tools.
- Invocation: StaticExecutor routes by tool name; wrap with ObservedExecutor for metrics, CancelableExecutor for ctx cancellation events, and ResilientExecutor for retries and circuit breaking.
- Policy: MemoryEvaluator evaluates ordered rules (fail-closed). RateLimitPerMinute applies per tenant+tool+rule window; returns RetryAfter when limited.
- Responses: build envelopes with Success/Error; attach trace IDs via WithTraceFromContext and request IDs via WithRequestID.
- Sessions: Postgres store uses pgx; unit tests mock pgx interfaces, no real DB required.

## Quick starts
- Load tools from file:
  - MCP_TOOLS_ROOT=/path/to/tools go test ./loader -run LoadFromFile
- Use registry with caching:
  - reg := registry.NewCachedRegistry(loader, 30*time.Second)
  - etag, tools, notMod, err := reg.ListToolsWithETag(ctx, "tenant", ifNoneMatch)
- Resilient execution:
  - exec := invoke.NewResilientExecutor(inner, invoke.ResilientConfig{MaxRetries: 2})
  - ch, err := exec.Invoke(ctx, req)

## Testing
- Run all tests with coverage:
  - make test-coverage (module root), or
  - go test ./... -cover
- CI/unit tests are self-contained; Postgres/Kafka are mocked via pgxmock/fakes.

See IMPLEMENTATION_GUIDE-en-US.md for integration patterns and extension points.
