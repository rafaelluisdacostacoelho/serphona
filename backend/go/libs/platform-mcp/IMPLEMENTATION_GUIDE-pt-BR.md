# IMPLEMENTATION_GUIDE (platform-mcp)

## Propósito
Como embutir os componentes do platform-mcp em servidores/clientes MCP respeitando os contratos do Serphona (multi-tenancy, tracing e envelopes de erro).

## Padrões principais
- **Carregamento de ferramentas**: use `loader.LoadFromFile(ctx, path, tenantID)` com `MCP_TOOLS_ROOT` definido. Caminhos são limpos, travessia é bloqueada e apenas arquivos regulares são aceitos. Schemas inválidos falham cedo via `protocol.ValidateTool`.
- **Registries**:
  - `MemoryRegistry` para testes/ephemera.
  - `CachedRegistry` encapsula qualquer loader com cache por TTL e helpers ETag em list/describe.
  - `AllowListRegistry` filtra por tenant a partir de um `AllowListProvider`; allow-list vazia é erro.
  - `PostgresLoader` lê ferramentas via pgx (`tenant_id` escopado). Combine com `CachedRegistry` em produção.
- **Invocação**:
  - `StaticExecutor` despacha por nome de ferramenta (case-insensitive).
  - Envolva com `ObservedExecutor` para métricas/auditoria e `CancelableExecutor` para cancelar via contexto.
  - Use `ResilientExecutor` para tentativas/backoff e breaker quando a execução falha antes de emitir eventos.
- **Políticas**: `policy.MemoryEvaluator` é fail-closed. Regras casam tenant/tool/ambiente/escopos; `RateLimitPerMinute` é por tenant+tool+rule e retorna `RetryAfter`.
- **Respostas**: construa envelopes `Success`/`Error`. Adicione correlação com `WithRequestID` e tracing com `WithTraceFromContext`. Paginação via `WithPagination`.
- **Sessões**: `session.PostgresStore` e `session.MemoryStore` cuidam do ciclo de vida da sessão; pgx é mockado em testes.

## Passos de integração (servidor típico)
1) Carregue manifestos de ferramentas na inicialização com `LoadFromFile` e faça upsert em um `MemoryRegistry` ou sirva via `PostgresLoader`+`CachedRegistry`.
2) Exponha list/describe usando métodos com ETag para honrar `If-None-Match` e reduzir payloads.
3) Monte a pilha de executor:
   - Base: `StaticExecutor` com handlers.
   - Wrap: `ObservedExecutor` (métricas/audit), `CancelableExecutor` (cancelamento), `ResilientExecutor` (retries/breaker).
4) Aplique acesso com `policy.MemoryEvaluator` em ordem de prioridade. Sem regra -> deny; retorne limites com `RetryAfter`.
5) Envolva respostas com `response.Success`/`response.Error`, incluindo trace/request IDs quando existirem.
6) Controle sessões (opcional) com `session.PostgresStore` para durabilidade ou `MemoryStore` em testes.

## Erros e observabilidade
- Loaders/registries propagam erros; providers de allow-list devem devolver falhas claramente.
- Caminho de invoke emite eventos de progresso e cancelamento; observers recebem duração de cada evento.
- Circuit breaker (`invoke.CircuitBreaker`) protege contra falhas repetidas e reabre após a janela.
- Use contexto OpenTelemetry; `WithTraceFromContext` puxa o trace ID ativo para os envelopes.

## Guia de testes
- Unit tests rápidos: `go test ./... -cover` (sem serviços externos).
- pgx é mockado com `pgxmock` e `fakeRows` nos testes de loader/store Postgres.
- Determinismo de tempo: CachedRegistry aceita `now` custom; CircuitBreaker expõe `now` para testes.

## Pontos de extensão
- Implemente `AllowListProvider` para conectar fontes externas de autorização.
- Forneça `Loader` custom para CachedRegistry (ex.: registry HTTP) implementando `ListTools`/`DescribeTool`.
- Implemente `Observer` para enviar métricas/auditoria para Prometheus/OTel.
- Troque a função de backoff ou de sleep no `ResilientExecutor` para ajustar retries.

## Restrições e segurança
- Não ignore `MCP_TOOLS_ROOT` ao ler do disco.
- Sempre valide ferramentas com `protocol.ValidateTool` antes de servi-las.
- Mantenha multi-tenancy intacto: registries e políticas exigem tenant IDs; não misture dados de tenants em caches.

## Exemplo mínimo de fiação
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
