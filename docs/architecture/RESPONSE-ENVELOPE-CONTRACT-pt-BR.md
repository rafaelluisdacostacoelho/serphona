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
- **Gin**: `response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, data, response.WithPagination(p))` e `response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "VALIDATION_ERROR", "invalid input", details)`.
- **chi/net/http**: no handler `response.WriteSuccess(r.Context(), w, http.StatusCreated, payload)`; para erros `response.WriteError(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "token missing", nil)`.
- **gRPC/metadata**: garanta interceptors inbound setando `x-request-id` e `traceparent`; clientes outbound devem propagar ambos via transporte instrumentado para que envelopes HTTP incluam os mesmos IDs.

## Checklist de migração
- Introduzir helpers compartilhados para Gin/chi/net/http para escrever sucesso/erro e injetar IDs.
- Adicionar testes em tabela por handler para travar a forma.
- Atualizar frontend/clientes para depender de `error.code` e `meta.pagination` apenas (sem formatos ad-hoc).
- Documentar códigos por serviço junto do OpenAPI.
