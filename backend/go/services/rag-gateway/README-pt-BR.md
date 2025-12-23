# rag-gateway

Gateway HTTP multi-tenant para operações de RAG (ingestão, consulta, namespaces). Usa pgvector como store e pode trabalhar com cliente de embedding noop ou OpenAI.

## Como rodar local
- Suba Postgres com a extensão pgvector e rode as migrações: `go run ./cmd/migrate`
- Inicie o gateway: `go run ./cmd/server`

## Configuração (variáveis de ambiente)
- HTTP_ADDR: endereço de escuta (padrão `:8086`).
- PGVECTOR_URL: DSN do Postgres (padrão `postgresql://postgres:postgres@localhost:5432/serphona_rag?sslmode=disable`).
- PGVECTOR_TABLE: nome da tabela (padrão `rag_chunks`).
- PGVECTOR_DIM: dimensão do embedding (padrão `1536`).
- PGVECTOR_LISTS: número de listas IVFFlat para o índice ANN; aplicado via GUC de sessão nas migrações (padrão `100`).
- RAG_TOPK_DEFAULT: top-k padrão quando o cliente não envia (padrão `5`).
- EMBEDDING_PROVIDER: `noop` ou `openai` (padrão `noop`).
- EMBEDDING_MODEL: nome do modelo de embedding (padrão `text-embedding-3-small`).
- EMBEDDING_API_KEY: obrigatório quando usar `openai`.
- EMBEDDING_BASE_URL: endpoint OpenAI customizado (opcional).

## Migrações
`go run ./cmd/migrate` aplica o SQL em ./migrations. Se precisar mudar o número de listas IVFFlat, defina `PGVECTOR_LISTS` antes de rodar; o comando de migrate envia via `PGOPTIONS` para que o bloco DO use o valor informado.

## Quickstart (exemplo OpenAI)
```
PGVECTOR_LISTS=200 go run ./cmd/migrate
EMBEDDING_PROVIDER=openai EMBEDDING_API_KEY=$SUA_CHAVE go run ./cmd/server
```
Depois faça POST em `POST /api/v1/query` com `tenant_id`, `namespace` e `query`.

### Exemplos de payload
Consulta
```
POST /api/v1/query
{
	"tenant_id": "tenant-123",
	"namespace": "support",
	"query": "como resetar a senha?",
	"top_k": 5,
	"filters": {"document_id": "doc-42"}
}
```

Ingestão
```
POST /api/v1/ingest
{
	"tenant_id": "tenant-123",
	"namespace": "support",
	"content": "Resete sua senha em Configurações > Segurança.",
	"metadata": {"document_id": "doc-42", "source": "faq"}
}
```

### Exemplos com curl
Ingestão (com header opcional `X-Tenant-ID` se você depender da inferência do middleware)
```
curl -X POST http://localhost:8086/api/v1/ingest \
	-H "Content-Type: application/json" \
	-H "X-Tenant-ID: tenant-123" \
	-d '{
		"tenant_id": "tenant-123",
		"namespace": "support",
		"content": "Resete sua senha em Configurações > Segurança.",
		"metadata": {"document_id": "doc-42", "source": "faq"}
	}'
```

Consulta (com header opcional `X-Tenant-ID`)
```
curl -X POST http://localhost:8086/api/v1/query \
	-H "Content-Type: application/json" \
	-H "X-Tenant-ID: tenant-123" \
	-d '{
		"tenant_id": "tenant-123",
		"namespace": "support",
		"query": "como resetar a senha?",
		"top_k": 3,
		"filters": {"document_id": "doc-42"}
	}'
```

### Override para Docker Compose (dev)
Use `backend/go/services/rag-gateway/deploy/docker-compose.override.yml` junto com o compose raiz para setar PGVECTOR/embedding e rodar migrações:
```
docker-compose -f docker-compose.yml \
	-f backend/go/services/rag-gateway/deploy/docker-compose.override.yml run --rm rag-gateway-migrate
docker-compose -f docker-compose.yml \
	-f backend/go/services/rag-gateway/deploy/docker-compose.override.yml up rag-gateway
```
