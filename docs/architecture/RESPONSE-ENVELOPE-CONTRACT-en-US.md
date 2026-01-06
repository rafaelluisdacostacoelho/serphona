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
- **Gin**
```go
func (h *Handler) Create(c *gin.Context) {
  var req createRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "invalid json", nil)
    return
  }
  data := h.svc.Create(c.Request.Context(), req)
  response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusCreated, data, response.WithPagination(response.Pagination{Page: 1, PageSize: 1, Total: 1, TotalPages: 1}))
}
```
- **chi / net/http**
```go
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
  id := chi.URLParam(r, "id")
  data, err := h.svc.Get(r.Context(), id)
  if err != nil {
    response.WriteError(r.Context(), w, http.StatusNotFound, "NOT_FOUND", "resource not found", nil)
    return
  }
  response.WriteSuccess(r.Context(), w, http.StatusOK, data)
}
```
- **gRPC unary** (ErrorInfo reason mirrors HTTP code; request/trace IDs come from interceptors)
```go
func (s *Server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
  data, err := s.svc.Get(ctx, req.Id)
  if err != nil {
    st := status.New(codes.NotFound, "resource not found")
    st, _ = st.WithDetails(&errdetails.ErrorInfo{Reason: "NOT_FOUND"})
    return nil, st.Err()
  }
  return &pb.GetResponse{Data: data}, nil
}
```
- **Outbound propagation**
  - HTTP clients: wrap transport with platform-auth client helpers so `x-request-id` and `traceparent` follow, and call `EnsureTenantHeader` when tenant is in context.
  - gRPC clients: include metadata `x-request-id`/`traceparent` (or use the provided interceptor) before invoking downstreams.

## Migration checklist
- Introduce shared helpers for Gin/chi/net/http to write success/error envelopes and inject IDs.
- Add table-driven tests per handler to lock the shape.
- Update frontend/clients to rely on `error.code` and `meta.pagination` only (no ad-hoc shapes).
- Document codes per service alongside OpenAPI.
