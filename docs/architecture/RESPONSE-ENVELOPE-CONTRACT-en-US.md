# Response Envelope Contract (HTTP + gRPC)

## Goals
- Consistent success/error shapes across Gin/chi/net/http and gRPC.
- Carry correlation IDs (request_id/trace_id) without leaking secrets.
- Make validation and pagination predictable for frontend/consumers.

## Success envelope (HTTP)
```json
{
  "data": { "...": "resource or array" },
  "meta": {
    "trace_id": "<trace-id>",
    "request_id": "<request-id>",
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 120,
      "total_pages": 6
    }
  }
}
```
Notes:
- `meta` is optional; include `trace_id`/`request_id` when available.
- For lists, populate `pagination`; omit for single-resource responses.
- Content-Type: application/json; charset=utf-8.

## Error envelope (HTTP)
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "field validation failed",
    "details": { "field": "reason" },
    "trace_id": "<trace-id>",
    "request_id": "<request-id>"
  }
}
```
Notes:
- `code` is short, stable, UPPER_SNAKE_CASE.
- `details` is optional; use for validation or domain context (no secrets/PII).
- Status codes: 4xx for client errors, 5xx for server.

## gRPC mapping
- Use `status.Status` codes for transport errors.
- Attach `trace_id`/`request_id` via metadata keys `traceparent` (W3C) and `x-request-id`.
- For rich errors, pack the HTTP error payload into `google.rpc.ErrDetails` (e.g., `ErrorInfo` with `reason=code` and `metadata` for details without secrets).
- Keep parity between HTTP `code` and gRPC `reason` values.

## Header/metadata sources
- `trace_id`: from tracer span context; use `traceparent` propagation.
- `request_id`: from platform-auth middleware; ensure outbound transport forwards it.
- Never include `Authorization`, cookies, or secrets in envelopes.

## Testing contracts
- HTTP: assert envelope shape, presence/absence of `meta.pagination`, and `trace_id` echo when middleware is enabled.
- gRPC: assert metadata contains `traceparent`/`x-request-id`; check `ErrorInfo.reason` matches expected code.

## Usage examples
- **Gin**: `response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, data, response.WithPagination(p))` and `response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", "invalid input", details)`.
- **chi/net/http**: inside a handler `response.WriteSuccess(r.Context(), w, http.StatusCreated, payload)`; on errors `response.WriteError(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "token missing", nil)`.
- **gRPC metadata**: ensure inbound interceptors set `x-request-id` and `traceparent`; outbound clients should propagate both via the instrumented transport so HTTP envelopes include the same IDs.

## Migration checklist
- Introduce shared helpers for Gin/chi/net/http to write success/error envelopes and inject IDs.
- Add table-driven tests per handler to lock the shape.
- Update frontend/clients to rely on `error.code` and `meta.pagination` only (no ad-hoc shapes).
- Document codes per service alongside OpenAPI.
