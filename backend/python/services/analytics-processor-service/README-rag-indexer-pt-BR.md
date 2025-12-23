# RAG Indexer (rascunho)

Trabalho esqueleto para consumir `rag.ingestion.requested`, buscar conteúdo, fazer chunking e upsert no pgvector.

Status: scaffold com fetch HTTP/S3 e upsert pg com embeddings (noop/OpenAI configurável).

## Como funciona (atual)
- Consumir o tópico Kafka `rag.ingestion.requested` (publisher com flag no rag-gateway).
- Validar payload com Pydantic (`RAGIngestionRequested`).
- Buscar blob via `uri` (HTTPS, S3/MinIO) com schemes permitidos; deduplicar com `etag`/If-None-Match.
- Fragmentar texto (~1,2k chars, overlap 80, limites naturais) e upsert em tabela pg com embeddings (coluna vector) e filtros de tenant/namespace.
- Comitar offsets após sucesso; DLQ/retry ainda não implementados.

## Rodando (dev stub)
```bash
cd backend/python/services/analytics-processor-service
python -m rag_indexer.main
```

## Próximos passos
- Adicionar retry/backoff + DLQ; cache de `etag` entre execuções.
- Reforçar RLS/políticas de tenant na conexão pg.
- Adicionar métricas (Prometheus) e tracing.
- Adicionar testes unitários mockando consumer/repo e pipeline de chunking.
