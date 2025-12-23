# platform-rag (esqueleto)

Propósito: componentes compartilhados de RAG (metadados, filtros, schemas, clientes) para serviços de ingestão, indexação e busca de conhecimento por tenant.

Status: somente esqueleto — sem código ainda; não afeta builds/tests.

Sugestão de conteúdo (próximos passos):
- `metadata/`: structs de metadados de documento/fragmento (tenant_id, namespace, idioma, canal, valid_from/valid_to, etag, acl/tags).
- `filters/`: helpers para impor filtros obrigatórios de tenant/canal/idioma.
- `retrieval/`: interfaces para clientes de vetor (pgvector/ClickHouse), rerankers e caches.
- `obs/`: hooks para tracing/métricas/logs (platform-observability).
- `test/`: contratos/mocks para vetorial e rerank.
