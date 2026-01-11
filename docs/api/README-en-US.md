# API Documentation (en-US)

This directory contains API specifications and documentation.

## Contents

- OpenAPI/Swagger specifications
- gRPC protobuf definitions
- API versioning guidelines
- MCP protocol contracts and tool catalog surfaces (RAG/MCP)

## Services (Go)

| Service | Type | Spec |
|---------|------|------|
| auth-gateway | REST | [openapi.yaml](./auth-gateway/openapi.yaml) |
| tenant-manager | REST | [openapi.yaml](./tenant-manager/openapi.yaml) |
| billing-service | REST | [openapi.yaml](./billing-service/openapi.yaml) |
| analytics-query-service | REST | [openapi.yaml](./analytics-query/openapi.yaml) |
| tools-manager | REST | Contract in backlog; OpenAPI pending |
| tools-gateway | REST | Contract tests/backlog; OpenAPI pending |
| agent-orchestrator | REST/gRPC | Spec pending (see service backlog) |
| voice-gateway | REST/gRPC | Spec pending (see service backlog) |
| rag-gateway | REST | Spec pending (RAG/MCP alignment) |
| platform-mcp (service) | MCP/gRPC | Spec pending (uses platform-mcp protocol) |

## Services (Python)

| Service | Type | Spec |
|---------|------|------|
| analytics-processor-service | Worker (Kafka/ClickHouse) | N/A (stream processor) |
| rag-processor-service | Worker (RAG pipelines) | N/A (stream processor) |
| reporting-export-service | Worker (exports) | N/A (batch/worker) |

## Tools / RAG / MCP surfaces

These surfaces back tool discovery/execution and MCP transport; detailed contracts live with each service/lib until formal OpenAPI/Proto specs are published.

| Service/Lib | Type | Spec / Reference |
|-------------|------|-------------------|
| tools-manager | REST | Contract in service backlog; API docs to-be-published. See [backend/go/services/tools-manager/BACKLOG.md](../../backend/go/services/tools-manager/BACKLOG.md). |
| tools-gateway | REST | Contract tests + backlog; OpenAPI pending. See [backend/go/services/tools-gateway/BACKLOG.md](../../backend/go/services/tools-gateway/BACKLOG.md). |
| rag-gateway | REST | See service backlog for RAG/MCP endpoints (spec pending). |
| platform-mcp (service) | MCP/gRPC | Uses platform-mcp protocol; backlog-driven contracts. |
| platform-mcp (library) | Library (MCP protocol) | MCP protocol/types and metrics. See [backend/go/libs/platform-mcp/BACKLOG.md](../../backend/go/libs/platform-mcp/BACKLOG.md). |

For end-to-end context and contracts across RAG/MCP, see [docs/architecture/RAG-MCP-en-US.md](../architecture/RAG-MCP-en-US.md). When OpenAPI/Proto specs are ready, add them here and link from the service READMEs.

## Libraries (Go)

| Library | Scope | Reference |
|---------|-------|-----------|
| platform-auth | Auth middleware/clients | [backend/go/libs/platform-auth](../../backend/go/libs/platform-auth) |
| platform-core | Core utilities | [backend/go/libs/platform-core](../../backend/go/libs/platform-core) |
| platform-events | Event schemas/utilities | [backend/go/libs/platform-events](../../backend/go/libs/platform-events) |
| platform-observability | Metrics/tracing/logging | [backend/go/libs/platform-observability](../../backend/go/libs/platform-observability) |
| platform-mcp | MCP protocol/client | [backend/go/libs/platform-mcp](../../backend/go/libs/platform-mcp) |
| platform-rag | RAG helpers/connectors | [backend/go/libs/platform-rag](../../backend/go/libs/platform-rag) |
