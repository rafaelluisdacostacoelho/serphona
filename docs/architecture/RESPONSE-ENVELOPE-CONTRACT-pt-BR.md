# Contrato de Envelope de Resposta (HTTP + gRPC)

## Objetivos
- Padronizar sucesso/erro em Gin/chi/net/http e gRPC.
- Levar IDs de correlação (request_id/trace_id) sem vazar segredos.
- Tornar validação e paginação previsíveis para frontend/consumidores.

## Envelope de sucesso (HTTP)
```json
{
  "data": { "...": "recurso ou lista" },
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
Observações:
- `meta` é opcional; inclua `trace_id`/`request_id` quando disponíveis.
- Para listas, preencha `pagination`; omita para recurso único.
- Content-Type: application/json; charset=utf-8.

## Envelope de erro (HTTP)
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "falha de validação",
    "details": { "campo": "motivo" },
    "trace_id": "<trace-id>",
    "request_id": "<request-id>"
  }
}
```
Observações:
- `code` curto, estável, em UPPER_SNAKE_CASE.
- `details` opcional; use para validação ou contexto de domínio (sem segredos/PII).
- Status: 4xx para erro de cliente, 5xx para servidor.

## Mapeamento gRPC
- Use códigos de `status.Status` para erros de transporte.
- Anexe `trace_id`/`request_id` em metadata `traceparent` (W3C) e `x-request-id`.
- Para erros ricos, coloque o payload de erro HTTP em `google.rpc.ErrDetails` (ex.: `ErrorInfo` com `reason=code` e `metadata` para detalhes sem segredos).
- Mantenha paridade entre `code` HTTP e `reason` gRPC.

## Fontes de header/metadata
- `trace_id`: do contexto de tracing; use propagação `traceparent`.
- `request_id`: do middleware platform-auth; garantir forward no transporte outbound.
- Nunca incluir `Authorization`, cookies ou segredos nos envelopes.

## Contratos de teste
- HTTP: validar forma do envelope, presença/ausência de `meta.pagination` e eco de `trace_id` quando middleware ativo.
- gRPC: validar metadata com `traceparent`/`x-request-id`; checar `ErrorInfo.reason` batendo com o código esperado.

## Exemplos de uso
- **Gin**
```go
func (h *Handler) Criar(c *gin.Context) {
  var req createRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", "json inválido", nil)
    return
  }
  data := h.svc.Create(c.Request.Context(), req)
  response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusCreated, data, response.WithPagination(response.Pagination{Page: 1, PageSize: 1, Total: 1, TotalPages: 1}))
}
```
- **chi / net/http**
```go
func (h *Handler) Obter(w http.ResponseWriter, r *http.Request) {
  id := chi.URLParam(r, "id")
  data, err := h.svc.Get(r.Context(), id)
  if err != nil {
    response.WriteError(r.Context(), w, http.StatusNotFound, "NOT_FOUND", "recurso não encontrado", nil)
    return
  }
  response.WriteSuccess(r.Context(), w, http.StatusOK, data)
}
```
- **gRPC unary** (ErrorInfo.reason espelha o código HTTP; request/trace IDs vêm dos interceptors)
```go
func (s *Server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
  data, err := s.svc.Get(ctx, req.Id)
  if err != nil {
    st := status.New(codes.NotFound, "recurso não encontrado")
    st, _ = st.WithDetails(&errdetails.ErrorInfo{Reason: "NOT_FOUND"})
    return nil, st.Err()
  }
  return &pb.GetResponse{Data: data}, nil
}
```
- **Propagação outbound**
  - Clientes HTTP: use os helpers de transporte do platform-auth para propagar `x-request-id` e `traceparent`, e chame `EnsureTenantHeader` quando houver tenant no contexto.
  - Clientes gRPC: inclua metadata `x-request-id`/`traceparent` (ou use o interceptor fornecido) antes de chamar serviços downstream.

## Checklist de migração
- Introduzir helpers compartilhados para Gin/chi/net/http para escrever sucesso/erro e injetar IDs.
- Adicionar testes em tabela por handler para travar a forma.
- Atualizar frontend/clientes para depender de `error.code` e `meta.pagination` apenas (sem formatos ad-hoc).
- Documentar códigos por serviço junto do OpenAPI.
