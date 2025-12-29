# IMPLEMENTATION_GUIDE (platform-mcp)

## Purpose
How to embed platform-mcp components into MCP servers/clients while respecting Serphona contracts (multi-tenancy, tracing, and error envelopes).

## Core patterns
- **Tool loading**: use `loader.LoadFromFile(ctx, path, tenantID)` with `MCP_TOOLS_ROOT` set. Paths are cleaned, traversal is blocked, and only regular files are accepted. Invalid schemas fail fast via `protocol.ValidateTool`.
- **Registries**:
  - `MemoryRegistry` for tests and ephemeral setups.
  - `CachedRegistry` wraps any loader with TTL-based cache and ETag-aware `List/Describe` helpers.
  - `AllowListRegistry` filters per-tenant based on an `AllowListProvider`; empty allow-list is an error.
  - `PostgresLoader` reads tools via pgx (`tenant_id` scoped). Compose it with `CachedRegistry` for production use.
- **Invocation**:
  - `StaticExecutor` dispatches to registered handlers by tool name (case-insensitive).
  - Wrap with `ObservedExecutor` to emit metrics/audit callbacks and `CancelableExecutor` to surface context cancellation as an error event.
  - Use `ResilientExecutor` for retry/backoff and optional circuit breaker when executor startup fails (errors before streaming events).
- **Policies**: `policy.MemoryEvaluator` is fail-closed. Rules match tenant/tool/env/scopes; `RateLimitPerMinute` is per tenant+tool+rule window and returns `RetryAfter`.
- **Responses**: build `Success`/`Error` envelopes. Add correlation with `WithRequestID` and tracing with `WithTraceFromContext`. Pagination via `WithPagination`.
- **Sessions**: `session.PostgresStore` and `session.MemoryStore` manage invocation session lifecycle; pgx is mocked in tests.

## Integration steps (typical server)
1) Load tool manifests at startup using `LoadFromFile` and upsert into a `MemoryRegistry` or directly serve through `PostgresLoader`+`CachedRegistry`.
2) Expose list/describe endpoints using the ETag-aware methods to honor `If-None-Match` and reduce payloads.
3) Build an executor stack:
   - Base: `StaticExecutor` with tool handlers.
   - Wrap: `ObservedExecutor` (metrics/audit), `CancelableExecutor` (ctx cancellation), `ResilientExecutor` (retries/breaker).
4) Enforce access via `policy.MemoryEvaluator` with ordered rules. Deny when no match; surface rate limits to clients.
5) Wrap responses with `response.Success`/`response.Error`, always including trace/request IDs when available.
6) Track sessions (optional) using `session.PostgresStore` for durability or `MemoryStore` for tests.

## Error handling and observability
- All loaders and registries return explicit errors; allow-list providers must surface failures (propagated to callers).
- Invoke path emits progress and cancellation events; observers receive the elapsed duration for each event.
- Circuit breaker (`invoke.CircuitBreaker`) guards repeated failures and auto-resets after the configured window.
- Use OpenTelemetry context to propagate traces; `WithTraceFromContext` pulls the active span’s trace ID into envelopes.

## Testing guidance
- Fast unit tests: `go test ./... -cover` (no external services required).
- pgx is mocked via `pgxmock` and `fakeRows` for Postgres loader/store tests.
- Deterministic time: CachedRegistry accepts a custom `now` func; CircuitBreaker exposes `now` for tests.

## Extension points
- Implement `AllowListProvider` to plug external authorization sources.
- Provide custom `Loader` for CachedRegistry (e.g., HTTP-based registry) by implementing `ListTools`/`DescribeTool`.
- Implement `Observer` to ship metrics/audit to Prometheus/OTel.
- Swap backoff function or sleep function in `ResilientExecutor` for custom retry behavior.

## Constraints and safety
- Never bypass `MCP_TOOLS_ROOT` checks when loading from disk.
- Always validate tools via `protocol.ValidateTool` before serving them.
- Keep multi-tenancy intact: registries and policies require tenant IDs; do not mix tenant data across caches.

## Example minimal wiring
```go
loader := registry.NewCachedRegistry(
    registry.NewPostgresLoader(pgxPool),
    30*time.Second,
)
allow := registry.NewAllowListRegistry(loader, myAllowProvider)
exec := invoke.NewResilientExecutor(
    invoke.NewObservedExecutor(
        invoke.NewCancelableExecutor(myStaticExecutor),
        myObserver,
    ),
    invoke.ResilientConfig{MaxRetries: 2},
)
```
