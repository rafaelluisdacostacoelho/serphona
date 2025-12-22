# Tools Gateway - Detailed Architecture (en-US)

> Full architecture for the Tools Gateway microservice.

## Index
1. Overview
2. High-Level Architecture
3. Clean Architecture
4. Data Flow
5. Core Components
6. Database Schema
7. Patterns and Decisions
8. Scalability
9. Security
10. Observability

---

## Overview
### Purpose
Tools Gateway abstracts integrations with external APIs by providing:
- Unified interface for diverse external APIs
- Input/output validation via JSON Schema
- Retry logic and circuit breakers for resilience
- Multi-tenancy with credential isolation
- Complete logging for analytics and billing
- Rate limiting and credit/usage control

### System Context
```
┌─────────────────────────────────────────────────────────┐
│                    Serphona Platform                    │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─────────────┐      ┌─────────────┐                   │
│  │   Agent     │─────▶│   Tools     │─────┐             │
│  │Orchestrator │      │   Gateway   │     │             │
│  └─────────────┘      └─────────────┘     │             │
│         │                    │             ▼             │
│         │                    │      ┌──────────────┐    │
│         │                    │      │  External    │    │
│         │                    └─────▶│  APIs        │    │
│         │                           │ (Google,     │    │
│         ▼                           │  OpenAI,     │    │
│  ┌─────────────┐                   │  Weather,    │    │
│  │  Analytics  │                   │  etc.)       │    │
│  │  Processor  │                   └──────────────┘    │
│  └─────────────┘                                        │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Responsibilities
1. Tool Registry (catalog)
2. Tool Execution (external calls)
3. Validation (JSON Schema input/output)
4. Authentication (credentials per tenant)
5. Resilience (retry, timeout, circuit breaker)
6. Observability (execution logging)
7. Billing (credit/usage tracking)

### MCP and Governed Catalog
- Tools Gateway feeds the MCP catalog (resources exposed as `resource = <namespace.action>`), keeping schemas, timeouts, and idempotency policy.
- Policy engine per tenant/agent/environment: allow/deny per tool, token scopes (platform-auth), rate limits and quotas per tenant.
- Standardized error contracts (code + message), logging with input/output hashes, latency, and status for audit/billing.
- RAG integration: `rag_query` tools can be exposed here to harmonize billing/observability.

---

## High-Level Architecture
(unchanged from pt-BR: component diagram, Clean Architecture layers)

## Clean Architecture
- Domain (entities: Tool, TenantTool, ToolExecution; interfaces: repositories, SchemaValidator, HTTPClient)
- Use Cases (ToolService, ToolExecutorService, TenantToolService)
- Infrastructure (PostgreSQL repos, HTTP client with retry, JSON Schema validator)
- Presentation (HTTP handlers/DTOs/middleware)

---

## Data Flow
- Execution flow: HTTP → auth → handler → ToolExecutorService → repositories → schema validation → HTTP client (with retry/auth/substitution) → schema validation → execution log → response.
- Creation flow: POST /tools → validate schemas → uniqueness → persist → DTO response.

---

## Core Components
- Domain entities and repositories (as in pt-BR version).
- ToolExecution logging: store input/output hashes, status, latency, credits, tenant/user IDs.

---

## Database Schema
- Tables with `tenant_id`, JSONB for schemas/config, RLS enabled; example DDL same as pt-BR version.

---

## Patterns and Decisions
- Clean Architecture; JSON Schema validation; retry/backoff; circuit breaker; rate limiting; credit accounting; MCP catalog alignment.

---

## Scalability
- Stateless API layer, horizontal scaling; per-tenant rate limits/quotas; connection pooling; async executor optional for long-running tools.

---

## Security
- platform-auth JWT with scopes; per-tenant credentials; secret storage via Vault/KMS; input/output validation to avoid injection; RLS on tables.

---

## Observability
- Tracing for each tool execution; metrics: success/error, latency, retries, rate-limit hits; logs with input/output hashes and tool resource.
