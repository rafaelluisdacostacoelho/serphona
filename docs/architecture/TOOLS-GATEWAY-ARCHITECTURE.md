# Tools Gateway - Arquitetura Detalhada

> 🔧 Documentação completa da arquitetura do microserviço Tools Gateway

## Índice

1. [Visão Geral](#visão-geral)
2. [Arquitetura de Alto Nível](#arquitetura-de-alto-nível)
3. [Clean Architecture](#clean-architecture)
4. [Fluxo de Dados](#fluxo-de-dados)
5. [Componentes Principais](#componentes-principais)
6. [Database Schema](#database-schema)
7. [Padrões e Decisões](#padrões-e-decisões)
8. [Escalabilidade](#escalabilidade)
9. [Segurança](#segurança)
10. [Observabilidade](#observabilidade)

---

## Visão Geral

### Propósito

O **Tools Gateway** é um microserviço que abstrai a complexidade de integrar com APIs de terceiros, fornecendo:

- Interface unificada para diferentes APIs externas
- Validação automática de entrada/saída via JSON Schema
- Retry logic e circuit breakers para resiliência
- Multi-tenancy com isolamento de credenciais
- Logging completo para analytics e billing
- Rate limiting e controle de consumo de créditos

### Contexto no Sistema

```
┌─────────────────────────────────────────────────────────┐
│                    Serphona Platform                     │
├─────────────────────────────────────────────────────────┤
│                                                           │
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
│                                                           │
└─────────────────────────────────────────────────────────┘
```

### Responsabilidades

1. **Tool Registry**: Gerenciar catálogo de ferramentas disponíveis
2. **Tool Execution**: Executar chamadas a APIs externas
3. **Validation**: Validar entrada e saída contra schemas JSON
4. **Authentication**: Gerenciar credenciais por tenant
5. **Resilience**: Retry logic, timeouts, circuit breakers
6. **Observability**: Logging de execuções para analytics
7. **Billing**: Rastreamento de consumo de créditos

---

## Arquitetura de Alto Nível

### Diagrama de Componentes

```
┌──────────────────────────────────────────────────────────────┐
│                     External Services                         │
│  (Google Search, OpenWeather, SendGrid, Custom APIs...)      │
└────────────────────────▲─────────────────────────────────────┘
                         │
                         │ HTTPS
                         │
┌────────────────────────┴─────────────────────────────────────┐
│                    Tools Gateway Service                      │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │                  Presentation Layer                       │ │
│ │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │ │
│ │  │  HTTP        │  │  DTOs        │  │  Middleware  │  │ │
│ │  │  Handlers    │  │              │  │  (Auth, etc) │  │ │
│ │  └──────────────┘  └──────────────┘  └──────────────┘  │ │
│ └──────────────────────────────────────────────────────────┘ │
│                          │                                    │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │                    Use Case Layer                         │ │
│ │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │ │
│ │  │    Tool      │  │     Tool     │  │   Tenant     │  │ │
│ │  │   Service    │  │   Executor   │  │    Tool      │  │ │
│ │  │              │  │   Service    │  │   Service    │  │ │
│ │  └──────────────┘  └──────────────┘  └──────────────┘  │ │
│ └──────────────────────────────────────────────────────────┘ │
│                          │                                    │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │                    Domain Layer                           │ │
│ │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │ │
│ │  │  Entities    │  │ Repositories │  │   Services   │  │ │
│ │  │  (Tool,      │  │ (Interfaces) │  │  (HTTP,      │  │ │
│ │  │   Execution) │  │              │  │   Validator) │  │ │
│ │  └──────────────┘  └──────────────┘  └──────────────┘  │ │
│ └──────────────────────────────────────────────────────────┘ │
│                          │                                    │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │                Infrastructure Layer                       │ │
│ │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │ │
│ │  │  PostgreSQL  │  │     HTTP     │  │    JSON      │  │ │
│ │  │  Repository  │  │    Client    │  │   Schema     │  │ │
│ │  │  (GORM)      │  │  (w/ retry)  │  │  Validator   │  │ │
│ │  └──────────────┘  └──────────────┘  └──────────────┘  │ │
│ └──────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │   PostgreSQL         │
              │   Database           │
              └──────────────────────┘
```

---

## Clean Architecture

### Camadas e Dependências

O serviço segue os princípios da Clean Architecture de Robert C. Martin:

```
┌─────────────────────────────────────────────────────────┐
│                                                          │
│    EXTERNAL                                              │
│    (APIs, DB, Web)                                       │
│                                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │                                                   │  │
│  │   INFRASTRUCTURE                                  │  │
│  │   (Repositories, HTTP Client, Validator)         │  │
│  │                                                   │  │
│  │  ┌────────────────────────────────────────────┐ │  │
│  │  │                                             │ │  │
│  │  │   PRESENTATION                              │ │  │
│  │  │   (HTTP Handlers, DTOs, Routes)            │ │  │
│  │  │                                             │ │  │
│  │  │  ┌──────────────────────────────────────┐ │ │  │
│  │  │  │                                       │ │ │  │
│  │  │  │   USE CASES                           │ │ │  │
│  │  │  │   (Business Logic)                    │ │ │  │
│  │  │  │                                       │ │ │  │
│  │  │  │  ┌────────────────────────────────┐ │ │ │  │
│  │  │  │  │                                 │ │ │ │  │
│  │  │  │  │   DOMAIN                        │ │ │ │  │
│  │  │  │  │   (Entities, Interfaces)        │ │ │ │  │
│  │  │  │  │                                 │ │ │ │  │
│  │  │  │  └────────────────────────────────┘ │ │ │  │
│  │  │  │                                       │ │ │  │
│  │  │  └──────────────────────────────────────┘ │ │  │
│  │  │                                             │ │  │
│  │  └────────────────────────────────────────────┘ │  │
│  │                                                   │  │
│  └──────────────────────────────────────────────────┘  │
│                                                          │
└─────────────────────────────────────────────────────────┘

Regra de Dependência: →→→ (de fora para dentro)
Camadas externas dependem de camadas internas
Camadas internas NÃO dependem de camadas externas
```

### 1. Domain Layer (Núcleo)

**Propósito**: Regras de negócio puras, independentes de frameworks

**Componentes**:

- **Entities**: `Tool`, `TenantTool`, `ToolExecution`
- **Repository Interfaces**: Contratos para acesso a dados
- **Service Interfaces**: `SchemaValidator`, `HTTPClient`

**Características**:
- Zero dependências externas
- Regras de negócio puras
- Testável isoladamente
- Imutável (não muda por mudanças em frameworks)

### 2. Use Case Layer (Aplicação)

**Propósito**: Orquestração de regras de negócio

**Componentes**:

- **ToolService**: CRUD de ferramentas com validações
- **ToolExecutorService**: Orquestração completa de execução
- **TenantToolService**: Gerenciamento de configurações tenant

**Responsabilidades**:
- Orquestrar entidades do domínio
- Coordenar repositories e services
- Implementar casos de uso específicos
- Validar regras de negócio compostas

### 3. Infrastructure Layer

**Propósito**: Implementações técnicas concretas

**Componentes**:

- **PostgreSQL Repositories**: Implementação GORM
- **HTTP Client**: Client com retry logic
- **Schema Validator**: Validação JSON Schema

**Características**:
- Implementa interfaces do domain
- Depende de frameworks externos
- Substituível sem afetar domínio

### 4. Presentation Layer (API)

**Propósito**: Interface HTTP REST

**Componentes**:

- **HTTP Handlers**: Controllers Gin
- **DTOs**: Request/Response objects
- **Middleware**: Autenticação, logging

**Características**:
- Transforma HTTP em casos de uso
- Valida entrada
- Formata saída
- Trata erros HTTP

---

## Fluxo de Dados

### Fluxo de Execução de Ferramenta

```
1. HTTP Request
   │
   ▼
2. [Gin Router] → Roteamento
   │
   ▼
3. [Auth Middleware] → Extrai tenant_id, user_id
   │
   ▼
4. [ToolHandler.ExecuteTool]
   │
   ├─ Valida request (DTO binding)
   │
   ▼
5. [ToolExecutorService.Execute]
   │
   ├─ 5.1 [ToolRepository] → Busca tool
   │      └─ Valida se está ativa
   │
   ├─ 5.2 [TenantToolRepository] → Busca config tenant
   │      ├─ Verifica se está habilitada
   │      └─ Valida permissões de usuário
   │
   ├─ 5.3 [SchemaValidator] → Valida input contra schema
   │
   ├─ 5.4 [HTTPClient.Execute] → Executa request HTTP
   │      ├─ Aplica autenticação (API Key, Bearer, Basic)
   │      ├─ Substitui placeholders na URL
   │      ├─ Adiciona query params/headers
   │      ├─ Executa com retry logic
   │      │   └─ Retry em: 5xx, 429, timeouts
   │      └─ Retorna response
   │
   ├─ 5.5 [SchemaValidator] → Valida output contra schema
   │
   ├─ 5.6 [ExecutionRepository] → Salva log de execução
   │      └─ Inclui: input, output, status, latency, credits
   │
   └─ 5.7 Retorna ExecutionResponse
      │
      ▼
6. [ToolHandler] → Formata response HTTP
   │
   ▼
7. HTTP Response
```

### Fluxo de Criação de Ferramenta

```
1. HTTP POST /api/v1/tools
   │
   ▼
2. [ToolHandler.CreateTool]
   │
   ├─ Valida request (DTO binding)
   ├─ Converte DTO → Entity
   │
   ▼
3. [ToolService.CreateTool]
   │
   ├─ 3.1 [SchemaValidator] → Valida input_schema
   ├─ 3.2 [SchemaValidator] → Valida output_schema
   ├─ 3.3 [ToolRepository] → Verifica unicidade do nome
   ├─ 3.4 Valida auth_type
   ├─ 3.5 Valida HTTP method
   ├─ 3.6 [ToolRepository] → Cria no DB
   │
   └─ Retorna Tool
      │
      ▼
4. [ToolHandler] → Converte Entity → DTO
   │
   ▼
5. HTTP 201 Created
```

---

## Componentes Principais

### 1. Domain Entities

#### Tool
```go
type Tool struct {
    ID                 uuid.UUID
    Name               string          // Unique identifier
    DisplayName        string          // Human-readable name
    Description        string
    Category           string          // search, email, weather, etc.
    Method             string          // GET, POST, PUT, DELETE, PATCH
    BaseURL            string          // https://api.example.com
    EndpointPath       string          // /v1/search
    Headers            json.RawMessage // Custom headers
    AuthType           string          // none, api_key, bearer, basic
    AuthConfig         json.RawMessage // Auth configuration
    InputSchema        json.RawMessage // JSON Schema for validation
    OutputSchema       json.RawMessage // JSON Schema for validation
    TimeoutSeconds     int             // Request timeout
    MaxRetries         int             // Retry attempts
    RetryDelaySeconds  int             // Delay between retries
    RateLimitPerMinute int             // Rate limit
    RateLimitPerHour   int
    CreditCost         int             // Credits per execution
    IsActive           bool            // Can be executed?
    IsPublic           bool            // Available to all tenants?
    CreatedAt          time.Time
    UpdatedAt          time.Time
}
```

#### TenantTool
```go
type TenantTool struct {
    ID                uuid.UUID
    TenantID          uuid.UUID
    ToolID            uuid.UUID
    CustomAuthConfig  json.RawMessage // Override auth config
    IsEnabled         bool
    AllowedUserIDs    []uuid.UUID     // User whitelist
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

#### ToolExecution
```go
type ToolExecution struct {
    ID              uuid.UUID
    TenantID        uuid.UUID
    UserID          uuid.UUID
    ToolID          uuid.UUID
    InputData       json.RawMessage
    OutputData      json.RawMessage
    Status          string          // success, error, timeout
    ErrorMessage    string
    LatencyMS       int
    CreditsConsumed int
    ExecutedAt      time.Time
}
```

### 2. Use Cases

#### ToolService

**Responsabilidades**:
- CRUD de ferramentas
- Validação de schemas JSON
- Verificação de unicidade
- Validação de tipos (auth, http method)

**Métodos principais**:
```go
type ToolService interface {
    CreateTool(ctx, *Tool) error
    GetTool(ctx, toolID) (*Tool, error)
    ListTools(ctx, filters) ([]*Tool, int64, error)
    UpdateTool(ctx, *Tool) error
    DeleteTool(ctx, toolID) error
}
```

#### ToolExecutorService

**Responsabilidades**:
- Orquestração completa de execução
- Validação de permissões
- Execução HTTP
- Logging para analytics
- Error handling

**Fluxo**:
1. Busca tool e valida status
2. Busca configuração tenant
3. Valida permissões de usuário
4. Valida input contra schema
5. Executa HTTP com retry
6. Valida output contra schema
7. Loga execução
8. Retorna resultado

**Métodos principais**:
```go
type ToolExecutorService interface {
    Execute(ctx, *ExecutionRequest) (*ExecutionResponse, error)
    GetExecution(ctx, executionID) (*ToolExecution, error)
    ListExecutions(ctx, tenantID, filters) ([]*ToolExecution, int64, error)
    GetExecutionStats(ctx, tenantID, from, to) (*ExecutionStats, error)
    GetCostBreakdown(ctx, tenantID, from, to) ([]*CostBreakdown, error)
}
```

#### TenantToolService

**Responsabilidades**:
- Configuração de ferramentas por tenant
- Gerenciamento de credenciais customizadas
- Controle de habilitação/desabilitação
- Permissões por usuário

**Métodos principais**:
```go
type TenantToolService interface {
    ConfigureTool(ctx, *TenantTool) error
    GetTenantTool(ctx, tenantID, toolID) (*TenantTool, error)
    ListTenantTools(ctx, tenantID, onlyEnabled) ([]*TenantTool, error)
    EnableTool(ctx, tenantID, toolID) error
    DisableTool(ctx, tenantID, toolID) error
}
```

### 3. Infrastructure Services

#### SchemaValidator

**Implementação**: `gojsonschema`

**Responsabilidades**:
- Validar JSON contra schemas
- Formatar erros de validação
- Verificar validade de schemas

**Métodos**:
```go
type SchemaValidator interface {
    ValidateInput(input, schema) error
    ValidateOutput(output, schema) error
    IsValidSchema(schema) error
}
```

#### HTTPClient

**Implementação**: `net/http` + `retry-go`

**Responsabilidades**:
- Executar requests HTTP
- Retry automático em erros temporários
- Timeout handling
- Autenticação (API Key, Bearer, Basic)
- URL placeholder replacement
- Query parameters e headers

**Retry Logic**:
- Retry em: timeouts, 5xx, 429
- Delay configurável entre retries
- Número máximo de tentativas configurável
- Backoff exponencial

**Métodos**:
```go
type HTTPClient interface {
    Execute(ctx, *Tool, input, authConfig) (*HTTPResponse, error)
}
```

---

## Database Schema

### Tools Table

```sql
CREATE TABLE tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    method VARCHAR(10) NOT NULL,
    base_url TEXT NOT NULL,
    endpoint_path TEXT NOT NULL,
    headers JSONB,
    auth_type VARCHAR(50) NOT NULL,
    auth_config JSONB,
    input_schema JSONB NOT NULL,
    output_schema JSONB NOT NULL,
    timeout_seconds INTEGER DEFAULT 30,
    max_retries INTEGER DEFAULT 3,
    retry_delay_seconds INTEGER DEFAULT 1,
    rate_limit_per_minute INTEGER DEFAULT 60,
    rate_limit_per_hour INTEGER DEFAULT 1000,
    credit_cost INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    is_public BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_tools_name ON tools(name);
CREATE INDEX idx_tools_category ON tools(category);
CREATE INDEX idx_tools_is_active ON tools(is_active);
```

### Tenant Tools Table

```sql
CREATE TABLE tenant_tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    tool_id UUID NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    custom_auth_config JSONB,
    is_enabled BOOLEAN DEFAULT true,
    allowed_user_ids UUID[],
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, tool_id)
);

CREATE INDEX idx_tenant_tools_tenant ON tenant_tools(tenant_id);
CREATE INDEX idx_tenant_tools_tool ON tenant_tools(tool_id);
CREATE INDEX idx_tenant_tools_enabled ON tenant_tools(is_enabled);
```

### Tool Executions Table

```sql
CREATE TABLE tool_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    tool_id UUID NOT NULL REFERENCES tools(id),
    input_data JSONB NOT NULL,
    output_data JSONB,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    latency_ms INTEGER,
    credits_consumed INTEGER DEFAULT 0,
    executed_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_executions_tenant ON tool_executions(tenant_id);
CREATE INDEX idx_executions_tool ON tool_executions(tool_id);
CREATE INDEX idx_executions_user ON tool_executions(user_id);
CREATE INDEX idx_executions_status ON tool_executions(status);
CREATE INDEX idx_executions_date ON tool_executions(executed_at DESC);
```

### Queries de Analytics

```sql
-- Total de execuções por ferramenta
SELECT 
    t.name,
    t.display_name,
    COUNT(*) as total_executions,
    COUNT(*) FILTER (WHERE e.status = 'success') as successful,
    COUNT(*) FILTER (WHERE e.status = 'error') as errors,
    AVG(e.latency_ms) as avg_latency_ms,
    SUM(e.credits_consumed) as total_credits
FROM tool_executions e
JOIN tools t ON e.tool_id = t.id
WHERE e.tenant_id = $1
  AND e.executed_at BETWEEN $2 AND $3
GROUP BY t.id, t.name, t.display_name
ORDER BY total_executions DESC;

-- Custo por dia
SELECT 
    DATE(executed_at) as date,
    SUM(credits_consumed) as daily_credits,
    COUNT(*) as daily_executions
FROM tool_executions
WHERE tenant_id = $1
  AND executed_at BETWEEN $2 AND $3
GROUP BY DATE(executed_at)
ORDER BY date DESC;
```

---

## Padrões e Decisões

### 1. Dependency Injection

**Decisão**: Usar constructor injection

**Razão**:
- Testabilidade (fácil criar mocks)
- Explícito (dependências claras)
- Type-safe (verificação em compile time)

**Exemplo**:
```go
func NewToolExecutorService(
    toolRepo repository.ToolRepository,
    tenantToolRepo repository.TenantToolRepository,
    executionRepo repository.ToolExecutionRepository,
    validator service.SchemaValidator,
    httpClient service.HTTPClient,
) ToolExecutorService {
    return &toolExecutorServiceImpl{
        toolRepo:       toolRepo,
        tenantToolRepo: tenantToolRepo,
        executionRepo:  executionRepo,
        validator:      validator,
        httpClient:     httpClient,
    }
}
```

### 2. Repository Pattern

**Decisão**: Interfaces no domain, implementações na infrastructure

**Razão**:
- Inversão de dependência
- Testabilidade (mocks)
- Substituibilidade (mudar de GORM para outro ORM)

### 3. Error Handling

**Estratégia**: Wrap errors com contexto

**Padrão**:
```go
if err := repo.Create(ctx, tool); err != nil {
    return fmt.Errorf("failed to create tool: %w", err)
}
```

### 4. JSON Schema Validation

**Decisão**: Usar JSON Schema padrão

**Razão**:
- Padrão da indústria
- Documentação auto-descritiva
- Reutilizável em outros contextos
- Suporte a validações complexas

### 5. Multi-tenancy

**Estratégia**: Tenant ID em todas operações

**Isolamento**:
- Tenant ID obrigatório em todas queries
- Credenciais isoladas por tenant
- Rate limits por tenant
- Billing por tenant

---

## Escalabilidade

### Horizontal Scaling

**Estratégia**: Stateless service

- Sem estado local
- Todas requisições independentes
- Database como única fonte de verdade
- Load balancer distribui tráfego

### Database Scaling

**Estratégias**:

1. **Read Replicas**: Separar leitura de escrita
2. **Connection Pooling**: GORM configurável
3. **Indexes**: Otimizar queries frequentes
4. **Partitioning**: Particionar por tenant_id ou data

### Caching (Futuro)

**Candidatos**:
- Tool metadata (raramente muda)
- Tenant configurations
- JSON Schemas compilados

**Tecnologia**: Redis

### Rate Limiting

**Implementação Atual**: Database-based (tool configuration)

**Futuro**: Redis-based distributed rate limiting

---

## Segurança

### Autenticação

**Atual**: Mock middleware (desenvolvimento)

**Produção**: JWT com validação via auth-gateway

### Autorização

**Modelo**: RBAC + Multi-tenancy

- Tenant isolation (tenant_id em todas operações)
- User whitelist por ferramenta (opcional)
- Tool-level permissions

### Credenciais Externas

**Armazenamento**: 
- Criptografadas no banco (TODO)
- Tenant-specific overrides
- Nunca expostas em logs

### Input Validation

**Camadas**:
1. HTTP binding validation (Gin)
2. Business logic validation (Use Cases)
3. JSON Schema validation (contra schemas definidos)

### Rate Limiting

**Proteções**:
- Por tool (configurável)
- Por tenant (futuro)
- Por usuário (futuro)

---

## Observability

### Logging

**Estratégia**: Structured logging

**Níveis**:
- INFO: Execuções bem-sucedidas
- WARN: Retries, rate limits
- ERROR: Falhas, validações

**Dados Logados**:
```json
{
  "execution_id": "uuid",
  "tenant_id": "uuid",
  "user_id": "uuid",
  "tool_id": "uuid",
  "tool_name": "google_search",
  "status": "success",
  "latency_ms": 245,
  "credits_consumed": 2,
  "timestamp": "2025-12-10T15:30:00Z"
}
```

### Metrics (Futuro)

**Prometheus**:
- `tool_executions_total{tool_id, status}`
- `tool_execution_duration_seconds{tool_id}`
- `tool_credits_consumed_total{tenant_id, tool_id}`
- `tool_rate_limit_hits_total{tenant_id}`

### Tracing (Futuro)

**OpenTelemetry**:
- End-to-end request tracing
- Identificar bottlenecks
- Debugging distribuído

### Health Checks

**Endpoint**: `GET /health`

**Verificações**:
- Database connectivity
- Disk space
- Memory usage

---

## Roadmap

### Fase 1: MVP ✅ (Concluído)
- [x] Clean Architecture
- [x] CRUD de ferramentas
- [x] Execução básica
- [x] JSON Schema validation
- [x] Retry logic
- [x] Multi-tenancy
- [x] Logging de execuções

### Fase 2: Production Hardening
- [ ] JWT authentication real
- [ ] Credenciais criptografadas
- [ ] Redis rate limiting
- [ ] Circuit breakers
- [ ] Kafka event publishing
- [ ] Prometheus metrics

### Fase 3: Advanced Features
- [ ] OAuth 2.0 support
- [ ] Webhook support
- [ ] Tool marketplace
- [ ] AI-powered tool discovery
- [ ] Auto-retry strategies
- [ ] Cost optimization

---

## Referências

- [Clean Architecture - Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [JSON Schema Specification](https://json-schema.org/)
- [The Twelve-Factor App](https://12factor.net/)
- [Go Clean Architecture Example](https://github.com/bxcodec/go-clean-arch)
- [Microservices Patterns](https://microservices.io/patterns/index.html)

---

## Contato

- **Repository**: https://github.com/rafaelluisdacostacoelho/serphona
- **Service**: tools-gateway
- **Documentation**: [README.md](../../backend/go/services/tools-gateway/README.md)
