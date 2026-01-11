# Documentação de API (pt-BR)

Este diretório contém especificações e documentação de APIs.

## Conteúdo

- Especificações OpenAPI/Swagger
- Definições protobuf gRPC
- Diretrizes de versionamento de APIs
- Contratos MCP e superfícies de catálogo de ferramentas (RAG/MCP)

## Serviços (Go)

| Serviço | Tipo | Spec |
|---------|------|------|
| auth-gateway | REST | [openapi.yaml](./auth-gateway/openapi.yaml) |
| tenant-manager | REST | [openapi.yaml](./tenant-manager/openapi.yaml) |
| billing-service | REST | [openapi.yaml](./billing-service/openapi.yaml) |
| analytics-query-service | REST | [openapi.yaml](./analytics-query/openapi.yaml) |
| tools-manager | REST | Contrato no backlog; OpenAPI pendente |
| tools-gateway | REST | Testes de contrato/backlog; OpenAPI pendente |
| agent-orchestrator | REST/gRPC | Spec pendente (ver backlog do serviço) |
| voice-gateway | REST/gRPC | Spec pendente (ver backlog do serviço) |
| rag-gateway | REST | Spec pendente (alinhamento RAG/MCP) |
| platform-mcp (serviço) | MCP/gRPC | Spec pendente (usa protocolo platform-mcp) |

## Serviços (Python)

| Serviço | Tipo | Spec |
|---------|------|------|
| analytics-processor-service | Worker (Kafka/ClickHouse) | N/A (processador de streams) |
| rag-processor-service | Worker (pipelines RAG) | N/A (processador de streams) |
| reporting-export-service | Worker (exports) | N/A (batch/worker) |

## Superfícies de Tools / RAG / MCP

Estas superfícies suportam descoberta/execução de ferramentas e transporte MCP; os contratos detalhados ficam em cada serviço/lib até que os OpenAPI/Proto formais sejam publicados.

| Serviço/Lib | Tipo | Spec / Referência |
|-------------|------|-------------------|
| tools-manager | REST | Contrato no backlog; API docs a publicar. Veja [backend/go/services/tools-manager/BACKLOG.md](../../backend/go/services/tools-manager/BACKLOG.md). |
| tools-gateway | REST | Testes de contrato + backlog; OpenAPI pendente. Veja [backend/go/services/tools-gateway/BACKLOG.md](../../backend/go/services/tools-gateway/BACKLOG.md). |
| rag-gateway | REST | Ver backlog do serviço para endpoints RAG/MCP (spec pendente). |
| platform-mcp (serviço) | MCP/gRPC | Usa protocolo platform-mcp; contratos guiados pelo backlog. |
| platform-mcp (biblioteca) | Biblioteca (protocolo MCP) | Protocolo/tipos e métricas MCP. Veja [backend/go/libs/platform-mcp/BACKLOG.md](../../backend/go/libs/platform-mcp/BACKLOG.md). |

Para contexto fim a fim e contratos de RAG/MCP, consulte [docs/architecture/RAG-MCP-pt-BR.md](../architecture/RAG-MCP-pt-BR.md). Quando os specs OpenAPI/Proto estiverem prontos, adicione aqui e referencie nos READMEs dos serviços.

## Bibliotecas (Go)

| Biblioteca | Escopo | Referência |
|------------|--------|------------|
| platform-auth | Middleware/clients de auth | [backend/go/libs/platform-auth](../../backend/go/libs/platform-auth) |
| platform-core | Utilitários centrais | [backend/go/libs/platform-core](../../backend/go/libs/platform-core) |
| platform-events | Esquemas/utilitários de eventos | [backend/go/libs/platform-events](../../backend/go/libs/platform-events) |
| platform-observability | Métricas/tracing/logging | [backend/go/libs/platform-observability](../../backend/go/libs/platform-observability) |
| platform-mcp | Protocolo/cliente MCP | [backend/go/libs/platform-mcp](../../backend/go/libs/platform-mcp) |
| platform-rag | Helpers/conectores RAG | [backend/go/libs/platform-rag](../../backend/go/libs/platform-rag) |
