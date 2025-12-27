# platform-rag

Blocos de RAG compartilhados para serviços multi-tenant (ingestão/indexação/busca).

O que já está utilizável:
- Modelos tipados para chunks/consultas com validação e defaults (`pkg/model`).
- Normalização/validação de metadados (`pkg/metadata`) — deduplica tags/ACL, espelha campos canônicos, valida TTL (0..1 ano) e sanidade básica de URI.
- Helper de chunking com overlap e cortes em limites naturais (`pkg/chunking`).
- Interface de vector store e implementação pgvector com helper de schema (`pkg/vector`).
- Adapters de embedding: cliente OpenAI com guardas de tokens, retries/backoff, checagem de dimensão; decorators de retry/cache em `pkg/embedding`.
- Decorator de rerank: envolva um `vector.Store` com `pkg/vector/reranker.Store` para reordenar resultados via reranker customizado.
- Filtros suportam campos de idioma/canal para consultas mais ricas.

Próximos/pendentes:
- Hooks de observabilidade (métricas/traces) nas operações do store.
- Adapters de clientes de embedding e decorators de filtros/rerank.

## Uso rápido

```go
cfg := pgvector.Config{
	URL:       os.Getenv("PG_DSN"),
	TableName: "rag_chunks",
	Dimension: 1536,
	Lists:     200,
}

db, _ := sql.Open("postgres", cfg.URL)
_ = pgvector.CreateSchema(context.Background(), db, cfg)

store := pgvector.NewStore(db, cfg)

chunk := model.Chunk{
	TenantID:   "tenant-1",
	Namespace:  "default",
	DocumentID: "doc-1",
	ChunkID:    "ch-1",
	Content:    "The quick brown fox",
	Embedding:  embeddingVector,
	Metadata:   model.ChunkMetadata{Source: "kb", Tags: []string{"faq"}},
}

_ = store.UpsertChunks(ctx, []model.Chunk{chunk})

q := model.Query{TenantID: "tenant-1", Namespace: "default", QueryVector: queryEmbedding, TopK: 5, MinScore: 0.2}
results, _ := store.Query(ctx, q)
```

### Teste de integração (opcional)
- Requer Postgres com extensão pgvector.
- Defina `TEST_PGVECTOR_DSN` e rode: `go test ./vector/pgvector -tags=integration`.
- Ou use o helper na raiz do repo: `make rag-pgvector-int` (sobe pgvector via docker-compose.tests.yml, roda a suíte de integração e depois para o container).

### Métricas (Prometheus)
- Crie um observer com `pgvector.NewPrometheusObserver(...)` e passe via `Config{Observer: obs}` para expor `platform_rag_pgvector_requests_total` e `platform_rag_pgvector_latency_seconds` com labels `operation` e `status` (ok|error).

### Embeddings
- Adapter OpenAI: use `NewOpenAIAdapter(client, tokenizer, OpenAIConfig{Model: ..., MaxTokens: ..., ExpectedDim: ...})`; aplica retries e valida tokens/dimensão.
- Decorators: `RetryableClient` para retries/backoff e `CachingClient` com cache em memória (TTL) via `NewInMemoryCache()`.

## Contrato de schema (pgvector)
- Tabela padrão `rag_chunks` (configurável).
- Colunas: tenant_id, namespace, document_id, chunk_id (PK), content, metadata (jsonb), embedding (vector), etag, created_at.
- Índices: PK em (tenant_id, namespace, chunk_id), ivfflat em embedding (lists configurável), lookup em (tenant_id, namespace, document_id).
- RLS: aplique políticas por tenant no Postgres em produção.

## Helper de chunking

```go
segments := chunking.ChunkText(texto, chunking.Options{MaxChars: 1200, Overlap: 80, NormalizeWhitespace: true, TrimSpace: true})
```

Defaults alinhados ao worker de ingestão (1200 chars, overlap 80, cortes em limites naturais). Use `chunking.DefaultOptions()` para uma configuração pré-definida.
`LanguageCode` pode orientar delimitadores de sentença para alguns idiomas (ex.: zh/ja usam 。！？) e `MaxSegments` limita o número de chunks em entradas muito longas.
Use `MaxSegments` para limitar o número de chunks em entradas muito longas.
