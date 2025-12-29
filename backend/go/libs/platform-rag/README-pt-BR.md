# platform-rag

Blocos de RAG compartilhados para serviços multi-tenant (ingestão, indexação, busca).

## O que tem aqui
- model/: Chunk, Query, Filter tipados com validação (tenant/ns obrigatórios, limites de TTL, TopK mínimo, dimensão de embedding >0).
- metadata/: normaliza/valida metadados (dedup tags/ACL, TTL 0..1 ano, sanidade de URI, chaves canônicas).
- chunking/: splits conscientes de limites com overlap; opções de max chars, overlap, hints de idioma, max segments, normalização de whitespace.
- embedding/: adapters (OpenAI) com guardas de tokens/dimensão e retries; decorators de retry/cache; interfaces para novos provedores.
- vector/: interface Store; store pgvector com helper de schema e observer Prometheus; decorator de rerank para reordenar resultados.

## Como usar
- Chunk e valide: use chunking.DefaultOptions(); valide chunks e metadados antes de persistir.
- Embeddings: escolha um cliente, opcionalmente envolva com RetryableClient e CachingClient.
- Store: configure pgvector (URL, TableName, Dimension, Lists, Observer); execute CreateSchema uma vez; use NewStore para UpsertChunks/Query.
- Filtros: suportam tags/acl/source/version/channel/language além de MinScore/TopK para controlar a busca.
- Rerank: envolva o store com reranker.Store para aplicar reranker externo preservando os scores vetoriais.

## Inícios rápidos
- Teste de integração (pgvector): defina TEST_PGVECTOR_DSN; rode `go test ./vector/pgvector -tags=integration` ou `make rag-pgvector-int`.
- Helper de chunking: `segments := chunking.ChunkText(texto, chunking.DefaultOptions())`.
- Métricas Prometheus: forneça observer via pgvector.Config{Observer: pgvector.NewPrometheusObserver()}.

## Testes
- Unit: go test ./...
- Integração: go test ./vector/pgvector -tags=integration (requer Postgres+pgvector).

Veja IMPLEMENTATION_GUIDE-pt-BR.md para padrões de integração detalhados.
