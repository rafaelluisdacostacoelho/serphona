# Tools Gateway Service

> 🔧 Microserviço para gerenciar e executar ferramentas externas (APIs) de forma centralizada, segura e escalável.

## 📋 Visão Geral

O **Tools Gateway** abstrai a complexidade de integrar com APIs de terceiros, fornecendo:

- ✅ Interface unificada para diferentes APIs
- ✅ Validação automática de entrada/saída via JSON Schema
- ✅ Retry logic e circuit breakers
- ✅ Multi-tenancy com isolamento de credenciais
- ✅ Logging completo para analytics
- ✅ Rate limiting por tenant
- ✅ Billing integration (consumo de créditos)

---

## 🚀 Quick Start

### Pré-requisitos

- Go 1.21+
- PostgreSQL 14+
- Make (opcional)

### Setup

```bash
# 1. Clone o repositório
git clone https://github.com/rafaelluisdacostacoelho/serphona.git
cd serphona/backend/go/services/tools-gateway

# 2. Instalar dependências
go mod tidy

# 3. Configurar banco de dados
createdb serphona_tools
psql serphona_tools < migrations/001_create_tools_tables.up.sql

# 4. Configurar variáveis de ambiente
cp .env.example .env
# Editar .env com suas configurações

# 5. Executar serviço
go run cmd/server/main.go
```

### Verificar Health

```bash
curl http://localhost:8085/health
```

---

## 📡 API Endpoints

### Tool Management

#### Criar Ferramenta

```bash
POST /api/v1/tools
Content-Type: application/json

{
  "name": "google_search",
  "display_name": "Google Search",
  "description": "Search the web using Google Custom Search API",
  "category": "search",
  "method": "GET",
  "base_url": "https://www.googleapis.com/customsearch/v1",
  "endpoint_path": "",
  "auth_type": "api_key",
  "auth_config": {
    "param_name": "key",
    "param_location": "query"
  },
  "input_schema": {
    "type": "object",
    "required": ["query"],
    "properties": {
      "query": {
        "type": "string",
        "description": "Search query"
      },
      "num": {
        "type": "integer",
        "minimum": 1,
        "maximum": 10,
        "default": 10
      }
    }
  },
  "output_schema": {
    "type": "object",
    "properties": {
      "items": {
        "type": "array"
      }
    }
  },
  "timeout_seconds": 30,
  "max_retries": 3,
  "credit_cost": 2
}
```

#### Listar Ferramentas

```bash
GET /api/v1/tools?category=search&limit=20&offset=0
```

**Response:**
```json
{
  "tools": [...],
  "total": 10,
  "limit": 20,
  "offset": 0
}
```

#### Obter Detalhes

```bash
GET /api/v1/tools/{tool_id}
```

### Tool Execution ⭐

#### Executar Ferramenta

```bash
POST /api/v1/tools/{tool_id}/execute
Content-Type: application/json
Authorization: Bearer {jwt_token}

{
  "input": {
    "query": "weather in São Paulo",
    "num": 5
  }
}
```

**Response:**
```json
{
  "execution_id": "550e8400-e29b-41d4-a716-446655440000",
  "tool_id": "123e4567-e89b-12d3-a456-426614174000",
  "tool_name": "google_search",
  "status": "success",
  "output": {
    "items": [
      {
        "title": "Weather in São Paulo",
        "link": "https://...",
        "snippet": "..."
      }
    ]
  },
  "latency_ms": 245,
  "credits_consumed": 2
}
```

---

## 🔧 Exemplo de Ferramentas

### 1. Google Search

```json
{
  "name": "google_search",
  "method": "GET",
  "base_url": "https://www.googleapis.com/customsearch/v1",
  "auth_type": "api_key",
  "auth_config": {
    "param_name": "key",
    "param_location": "query"
  }
}
```

### 2. OpenWeather

```json
{
  "name": "weather_current",
  "method": "GET",
  "base_url": "https://api.openweathermap.org/data/2.5/weather",
  "auth_type": "api_key",
  "auth_config": {
    "param_name": "appid",
    "param_location": "query"
  }
}
```

### 3. SendGrid Email

```json
{
  "name": "sendgrid_email",
  "method": "POST",
  "base_url": "https://api.sendgrid.com/v3/mail/send",
  "auth_type": "bearer",
  "auth_config": {
    "header_name": "Authorization",
    "prefix": "Bearer"
  }
}
```

---

## 🏗️ Arquitetura

### Clean Architecture

```
┌─────────────────────────────────────┐
│       Presentation Layer            │
│  (HTTP Handlers, DTOs, Routes)      │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│         Use Case Layer              │
│  (Business Logic, Orchestration)    │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│         Domain Layer                │
│  (Entities, Repository Interfaces)  │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│      Infrastructure Layer           │
│  (Database, HTTP Client, Validator) │
└─────────────────────────────────────┘
```

### Componentes Principais

#### Domain Layer
- **Entities**: Tool, TenantTool, ToolExecution
- **Repositories**: Interfaces para acesso a dados
- **Services**: Validator, HTTPClient

#### Use Case Layer
- **ToolService**: CRUD de ferramentas
- **ToolExecutorService**: Execução de ferramentas
- **TenantToolService**: Configurações por tenant

#### Infrastructure Layer
- **PostgreSQL Repositories**: Implementação GORM
- **Schema Validator**: JSON Schema validation
- **HTTP Client**: Client com retry logic

#### Presentation Layer
- **HTTP Handlers**: Controllers Gin
- **DTOs**: Request/Response objects

---

## 🔐 Autenticação

### API Key

```json
{
  "auth_type": "api_key",
  "auth_config": {
    "api_key": "your-api-key",
    "param_name": "api_key",
    "param_location": "query"
  }
}
```

### Bearer Token

```json
{
  "auth_type": "bearer",
  "auth_config": {
    "token": "your-bearer-token",
    "header_name": "Authorization",
    "prefix": "Bearer"
  }
}
```

### Basic Auth

```json
{
  "auth_type": "basic",
  "auth_config": {
    "username": "user",
    "password": "pass"
  }
}
```

---

## 🔄 Fluxo de Execução

```
1. Request recebido
   ↓
2. Autenticação JWT (extrai tenant_id, user_id)
   ↓
3. Busca ferramenta no DB
   ↓
4. Valida se está ativa
   ↓
5. Busca configuração tenant (se existir)
   ↓
6. Valida permissões de usuário
   ↓
7. VALIDA INPUT contra JSON Schema
   ↓
8. EXECUTA HTTP com retry logic
   ↓
9. VALIDA OUTPUT contra JSON Schema
   ↓
10. LOGA execução no DB (analytics)
    ↓
11. Retorna resultado
```

---

## 🗄️ Database Schema

### Tools Table

```sql
CREATE TABLE tools (
    id UUID PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    method VARCHAR(10) NOT NULL,
    base_url TEXT NOT NULL,
    endpoint_path TEXT NOT NULL,
    auth_type VARCHAR(50) NOT NULL,
    auth_config JSONB,
    input_schema JSONB NOT NULL,
    output_schema JSONB NOT NULL,
    timeout_seconds INTEGER DEFAULT 30,
    max_retries INTEGER DEFAULT 3,
    rate_limit_per_minute INTEGER DEFAULT 60,
    credit_cost INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    is_public BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Tenant Tools Table

```sql
CREATE TABLE tenant_tools (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    tool_id UUID NOT NULL REFERENCES tools(id),
    custom_auth_config JSONB,
    is_enabled BOOLEAN DEFAULT true,
    allowed_user_ids UUID[],
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Tool Executions Table

```sql
CREATE TABLE tool_executions (
    id UUID PRIMARY KEY,
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
```

---

## ⚙️ Configuração

### Variáveis de Ambiente

```bash
# Server
SERVER_PORT=8085
ENV=development

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/serphona_tools

# Timeouts
TOOL_TIMEOUT=30s
TOOL_MAX_RETRIES=3

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=100

# Billing
ENABLE_CREDIT_CONSUMPTION=true
CREDIT_COST_PER_TOOL_CALL=1
```

---

## 📊 Observability

### Metrics (TODO)

- `tool_executions_total{tool_id, status}`
- `tool_execution_duration_seconds{tool_id}`
- `tool_credits_consumed_total{tenant_id, tool_id}`
- `tool_rate_limit_hits_total{tenant_id}`

### Logging

Todas as execuções são logadas em `tool_executions` para analytics.

---

## 🧪 Testes

```bash
# Unit tests
go test ./internal/...

# Integration tests
go test ./internal/adapter/http/handler/... -tags=integration

# Coverage
go test -cover ./...
```

---

## 🚢 Deploy

### Docker

```bash
docker build -t tools-gateway:latest .
docker run -p 8085:8085 --env-file .env tools-gateway:latest
```

### Kubernetes

```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

---

## 📝 Roadmap

- [ ] OAuth 2.0 support
- [ ] Rate limiting com Redis
- [ ] Kafka event publishing
- [ ] Prometheus metrics
- [ ] OpenTelemetry tracing
- [ ] Circuit breakers
- [ ] Tool marketplace
- [ ] Webhook support

---

## 🤝 Contributing

1. Fork o projeto
2. Crie uma branch (`git checkout -b feature/amazing`)
3. Commit suas mudanças (`git commit -m 'Add amazing feature'`)
4. Push para a branch (`git push origin feature/amazing`)
5. Abra um Pull Request

---

## 📄 License

Este projeto é parte da plataforma Serphona.

---

## 👥 Autores

- **Serphona Team** - [GitHub](https://github.com/rafaelluisdacostacoelho/serphona)

---

## 📚 Documentação Adicional

- [PLANNING.md](./PLANNING.md) - Documento de planejamento detalhado
- [Architecture Docs](../../docs/architecture/)
- [API Specification](../../docs/api/)
