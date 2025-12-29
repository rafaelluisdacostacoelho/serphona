# IMPLEMENTATION_GUIDE (platform-rag)

## Propósito
Como embutir os componentes do platform-rag (chunking, validação de metadados, embeddings, vector stores) em serviços multi-tenant mantendo schemas, observabilidade e tenancy.

## Componentes centrais
- **Modelos** (`model`): `Chunk`, `Query`, `Filter` tipados com validação (tenant/namespace obrigatórios, limites de TTL, TopK mínimo, dimensão de embedding >0).
- **Metadados** (`metadata`): normaliza/valida metadata de chunks (dedup tags/ACL, TTL 0..1 ano, checagem básica de URI, chaves canônicas).
- **Chunking** (`chunking`): splits conscientes de limites com overlap; `Options{MaxChars, Overlap, LanguageCode, MaxSegments, NormalizeWhitespace, TrimSpace}`; defaults alinhados ao worker (1200 chars, overlap 80).
- **Embeddings** (`embedding`): adapter OpenAI com guardas de tokens/dimensão e retries; decorators de retry/cache em memória; interfaces para novos provedores.
- **Vector store** (`vector`): interface `Store` (UpsertChunks, Query); implementação pgvector + helper `pgvector.CreateSchema` e observer Prometheus; decorator de rerank para reordenar resultados.

## Passos de integração (fluxo de servidor)
1) **Chunk e valida**: use `chunking.ChunkText` ou `ChunkDocument` com `chunking.DefaultOptions()`; valide chunks com `model.ValidateChunk` e metadata com `metadata.Normalize`/`Validate`.
2) **Gerar embedding**: escolha um cliente (ex.: `embedding.OpenAIAdapter`), opcionalmente envolva com `RetryableClient` e `CachingClient` para resiliência/custo.
3) **Store**: configure pgvector `Config{URL, TableName, Dimension, Lists, Observer}`; execute `pgvector.CreateSchema(ctx, db, cfg)`; depois `store := pgvector.NewStore(db, cfg)`.
4) **Upsert/query**: chame `store.UpsertChunks(ctx, chunks)` e `store.Query(ctx, query)`; inclua `Filter` (tags/acl/source/version/channel/language) e `MinScore/TopK` para moldar resultados.
5) **Rerank (opcional)**: envolva o store com `vector/reranker.Store` para reordenar resultados via reranker externo preservando o score vetorial.
6) **Observabilidade**: habilite o observer Prometheus no config para emitir métricas de requisição/latência por operação/status.

## Tenancy e segurança
- Sempre defina `TenantID` e `Namespace` em chunks e queries; aplique RLS no Postgres em produção.
- Nome de tabela: use tabelas únicas por ambiente/namespace se necessário; `CreateSchema` valida nomes.
- TTL de chunk: limitado a um ano; zero desativa expiração. Aplique TTL via jobs de retenção ou políticas do DB.

## Tratamento de erros
- Adapters de embedding retornam erros tipados para tokens/dimensão e propagam erros do provedor.
- Operações do vector store retornam erros diretamente; se preciso, reforce com retries no chamador.
- O decorator de rerank devolve erros do store sem alteração.

## Testes
- Unit: `go test ./...` (interações pgx/DB são mockadas onde necessário).
- Integração (pgvector): defina `TEST_PGVECTOR_DSN` e rode `go test ./vector/pgvector -tags=integration` ou `make rag-pgvector-int` (usa docker-compose.tests.yml).
- Chunking/metadata: testes cobrem condições de borda (overlap, MaxSegments, limites de TTL, sanidade de URI).

## Pontos de extensão
- Implemente novos provedores de embedding satisfazendo `embedding.Client`.
- Adicione observabilidade com seu próprio `vector.Observer`.
- Troque a estratégia de rerank fornecendo um novo reranker ao `reranker.Store`.

## Exemplo de fiação
```go
cfg := pgvector.Config{URL: dsn, TableName: "rag_chunks", Dimension: 1536, Lists: 200, Observer: pgvector.NewPrometheusObserver()}
_ = pgvector.CreateSchema(ctx, db, cfg)
store := pgvector.NewStore(db, cfg)
emb := embedding.NewOpenAIAdapter(client, tokenizer, embedding.OpenAIConfig{Model: "text-embedding-3-large", MaxTokens: 8192, ExpectedDim: 1536})
chunks := chunking.ChunkText(texto, chunking.DefaultOptions())
_ = store.UpsertChunks(ctx, chunks)
res, _ := store.Query(ctx, model.Query{TenantID: "t1", Namespace: "default", QueryVector: vec, TopK: 5, MinScore: 0.2})
```
