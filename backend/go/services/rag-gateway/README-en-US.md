# rag-gateway

Tenant-scoped HTTP gateway for RAG operations (ingest, query, namespaces). Uses pgvector as the backing store and can call either a noop embedding client or OpenAI embeddings.

## Run locally
- Start Postgres with pgvector extension and run migrations: `go run ./cmd/migrate`
- Start the gateway: `go run ./cmd/server`

## Configuration (env vars)
- HTTP_ADDR: listen address (default `:8086`).
- PGVECTOR_URL: Postgres DSN (default `postgresql://postgres:postgres@localhost:5432/serphona_rag?sslmode=disable`).
- PGVECTOR_TABLE: table name (default `rag_chunks`).
- PGVECTOR_DIM: embedding dimension (default `1536`).
- PGVECTOR_LISTS: IVFFlat list count for the ANN index; applied via session GUC during migrations (default `100`).
- RAG_TOPK_DEFAULT: default top-k when the client omits it (default `5`).
- EMBEDDING_PROVIDER: `noop` or `openai` (default `noop`).
- EMBEDDING_MODEL: embedding model name (default `text-embedding-3-small`).
- EMBEDDING_API_KEY: required when using `openai`.
- EMBEDDING_BASE_URL: override OpenAI base URL (optional).

## Migrations
`go run ./cmd/migrate` applies SQL in ./migrations. If you need a different IVFFlat list count, set `PGVECTOR_LISTS` before running; the migrate command passes it via `PGOPTIONS` so the DO block in the migration uses your value.

## Quickstart (OpenAI example)
```
PGVECTOR_LISTS=200 go run ./cmd/migrate
EMBEDDING_PROVIDER=openai EMBEDDING_API_KEY=$YOUR_KEY go run ./cmd/server
```
Then POST to `POST /api/v1/query` with `tenant_id`, `namespace`, and `query`.

### Example payloads
Query
```
POST /api/v1/query
{
	"tenant_id": "tenant-123",
	"namespace": "support",
	"query": "how to reset password?",
	"top_k": 5,
	"filters": {"document_id": "doc-42"}
}
```

Ingest
```
POST /api/v1/ingest
{
	"tenant_id": "tenant-123",
	"namespace": "support",
	"content": "Reset your password in Settings > Security.",
	"metadata": {"document_id": "doc-42", "source": "faq"}
}
```

### curl examples
Ingest (with optional header `X-Tenant-ID` if you rely on middleware inference)
```
curl -X POST http://localhost:8086/api/v1/ingest \
	-H "Content-Type: application/json" \
	-H "X-Tenant-ID: tenant-123" \
	-d '{
		"tenant_id": "tenant-123",
		"namespace": "support",
		"content": "Reset your password in Settings > Security.",
		"metadata": {"document_id": "doc-42", "source": "faq"}
	}'
```

Query (with optional header `X-Tenant-ID`)
```
curl -X POST http://localhost:8086/api/v1/query \
	-H "Content-Type: application/json" \
	-H "X-Tenant-ID: tenant-123" \
	-d '{
		"tenant_id": "tenant-123",
		"namespace": "support",
		"query": "how to reset password?",
		"top_k": 3,
		"filters": {"document_id": "doc-42"}
	}'
```

### Docker Compose override (dev)
Use `backend/go/services/rag-gateway/deploy/docker-compose.override.yml` alongside the root compose to set PGVECTOR/embedding knobs and run migrations:
```
docker-compose -f docker-compose.yml \
	-f backend/go/services/rag-gateway/deploy/docker-compose.override.yml run --rm rag-gateway-migrate
docker-compose -f docker-compose.yml \
	-f backend/go/services/rag-gateway/deploy/docker-compose.override.yml up rag-gateway
```
