# Tools Gateway - Planning & Architecture Document

> 📋 Documento de planejamento e arquitetura para implementação completa do Tools Gateway

## 📌 Visão Geral

O **Tools Gateway** é um microserviço responsável por gerenciar e executar ferramentas externas (external APIs) de forma centralizada, segura e escalável. Ele abstrai a complexidade de integrar com APIs de terceiros, fornecendo uma interface unificada para agentes de IA e outros serviços da plataforma Serphona.

### Objetivos Principais

1. **Abstração de APIs Externas**: Fornecer interface unificada para diferentes APIs (REST, GraphQL, etc)
2. **Segurança Multi-tenant**: Isolamento de credenciais e configurações por tenant
3. **Validação de Schemas**: Validar entrada/saída contra schemas JSON
4. **Rate Limiting**: Controlar uso por tenant e por ferramenta
5. **Observabilidade**: Logging completo para analytics e debugging
6. **Billing Integration**: Consumo de créditos por execução de ferramenta
7. **Resiliência**: Retry logic, circuit breakers, timeouts
8. **Extensibilidade**: Fácil adição de novas ferramentas

---

## 🏗️ Arquitetura

### Componentes Principais

```
┌─────────────────────────────────────────────────────────┐
│                  Tools Gateway Service                  │
│                                                         │
│  ┌──────────────┐  ┌───────────────┐  ┌──────────────┐  │
│  │ Tool Registry│  │   Executor    │  │  Validator   │  │
│  │              │  │               │  │              │  │
│  │ • CRUD Tools │  │ • HTTP Client │  │ • JSON Schema│  │
│  │ • Schemas    │  │ • Retry Logic │  │ • Input/Out  │  │
│  │ • Auth Cfg   │  │ • Timeouts    │  │ • Rate Limit │  │
│  └──────────────┘  └───────────────┘  └──────────────┘  │
│                                                         │
│  ┌──────────────┐  ┌───────────────┐  ┌──────────────┐  │
│  │ Auth Manager │  │ Rate Limiter  │  │   Analytics  │  │
│  │              │  │               │  │              │  │
│  │ • Per-Tool   │  │ • Redis Based │  │ • Kafka Pub  │  │
│  │ • Per-Tenant │  │ • Token Bucket│  │ • Latency    │  │
│  │ • OAuth Flow │  │ • Per Tenant  │  │ • Success    │  │
│  └──────────────┘  └───────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
   ┌────────────┐       ┌────────────┐       ┌───────────┐
   │ PostgreSQL │       │   Redis    │       │   Kafka   │
   │            │       │            │       │           │
   │  Tools DB  │       │ Rate Limit │       │ Analytics │
   └────────────┘       └────────────┘       └───────────┘
```

### Fluxo de Execução

```
1. Agent/Service Request
   POST /api/v1/tools/google_search/execute
   {
     "query": "weather in São Paulo",
     "num_results": 5
   }

2. Authentication (JWT)
   → Extract tenant_id, user_id
   → Validate permissions

3. Tool Lookup
   → Fetch tool config from DB
   → Validate tool exists
   → Check tenant has access

4. Input Validation
   → Validate against input_schema
   → Transform parameters

5. Rate Limiting
   → Check tenant quota
   → Check tool-specific limits

6. Credit Check (Billing)
   → Verify tenant has credits
   → Calculate cost

7. Tool Execution
   → Prepare HTTP request
   → Add authentication (API key, OAuth, etc)
   → Execute with retry logic
   → Handle timeouts

8. Output Validation
   → Validate response against output_schema
   → Transform response

9. Analytics & Billing
   → Publish to Kafka
   → Deduct credits
   → Log execution metrics

10. Response
    {
      "tool_id": "google_search",
      "status": "success",
      "data": {...},
      "latency_ms": 245,
      "credits_consumed": 1
    }
```

---

## 💾 Data Model

### Tools Table

```sql
CREATE TABLE tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100), -- search, communication, weather, calendar, etc
    
    -- HTTP Configuration
    method VARCHAR(10) NOT NULL, -- GET, POST, PUT, DELETE, PATCH
    base_url TEXT NOT NULL,
    endpoint_path TEXT NOT NULL,
    headers JSONB DEFAULT '{}',
    
    -- Authentication
    auth_type VARCHAR(50) NOT NULL, -- none, api_key, oauth2, bearer
    auth_config JSONB DEFAULT '{}', -- depends on auth_type
    
    -- Schemas
    input_schema JSONB NOT NULL, -- JSON Schema for input validation
    output_schema JSONB NOT NULL, -- JSON Schema for output validation
    
    -- Configuration
    timeout_seconds INTEGER DEFAULT 30,
    max_retries INTEGER DEFAULT 3,
    retry_delay_seconds INTEGER DEFAULT 1,
    
    -- Rate Limiting
    rate_limit_per_minute INTEGER DEFAULT 60,
    rate_limit_per_hour INTEGER DEFAULT 1000,
    
    -- Billing
    credit_cost INTEGER DEFAULT 1, -- credits consumed per execution
    
    -- Metadata
    is_active BOOLEAN DEFAULT true,
    is_public BOOLEAN DEFAULT false, -- available to all tenants
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    
    -- Indexes
    CONSTRAINT tools_name_key UNIQUE (name)
);

CREATE INDEX idx_tools_category ON tools(category);
CREATE INDEX idx_tools_is_active ON tools(is_active);
CREATE INDEX idx_tools_is_public ON tools(is_public);
```

### Tenant Tools (Custom/Private Tools)

```sql
CREATE TABLE tenant_tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tool_id UUID NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    
    -- Override configurations
    custom_auth_config JSONB, -- tenant-specific API keys, OAuth tokens
    custom_rate_limit_per_minute INTEGER,
    custom_credit_cost INTEGER,
    
    -- Permissions
    is_enabled BOOLEAN DEFAULT true,
    allowed_user_ids UUID[], -- if set, only these users can use
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT tenant_tools_tenant_tool_key UNIQUE (tenant_id, tool_id)
);

CREATE INDEX idx_tenant_tools_tenant_id ON tenant_tools(tenant_id);
CREATE INDEX idx_tenant_tools_tool_id ON tenant_tools(tool_id);
```

### Tool Executions (Analytics)

```sql
CREATE TABLE tool_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    user_id UUID REFERENCES users(id),
    tool_id UUID NOT NULL REFERENCES tools(id),
    
    -- Request/Response
    input_data JSONB NOT NULL,
    output_data JSONB,
    
    -- Execution Details
    status VARCHAR(50) NOT NULL, -- success, error, timeout, rate_limited
    error_message TEXT,
    latency_ms INTEGER,
    
    -- Billing
    credits_consumed INTEGER DEFAULT 0,
    
    -- Metadata
    executed_at TIMESTAMP DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_tool_executions_tenant_id ON tool_executions(tenant_id);
CREATE INDEX idx_tool_executions_tool_id ON tool_executions(tool_id);
CREATE INDEX idx_tool_executions_executed_at ON tool_executions(executed_at);
CREATE INDEX idx_tool_executions_status ON tool_executions(status);
```

---

## 🔧 Tool Types & Examples

### 1. Search Tools

#### Google Custom Search

```json
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
        "default": 10,
        "description": "Number of results"
      }
    }
  },
  "output_schema": {
    "type": "object",
    "properties": {
      "items": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "title": {"type": "string"},
            "link": {"type": "string"},
            "snippet": {"type": "string"}
          }
        }
      }
    }
  },
  "credit_cost": 2
}
```

### 2. Weather Tools

#### OpenWeather API

```json
{
  "name": "weather_current",
  "display_name": "Current Weather",
  "description": "Get current weather for a location",
  "category": "weather",
  "method": "GET",
  "base_url": "https://api.openweathermap.org/data/2.5/weather",
  "endpoint_path": "",
  "auth_type": "api_key",
  "auth_config": {
    "param_name": "appid",
    "param_location": "query"
  },
  "input_schema": {
    "type": "object",
    "required": ["location"],
    "properties": {
      "location": {
        "type": "string",
        "description": "City name or coordinates"
      },
      "units": {
        "type": "string",
        "enum": ["metric", "imperial", "standard"],
        "default": "metric"
      }
    }
  },
  "output_schema": {
    "type": "object",
    "properties": {
      "temperature": {"type": "number"},
      "feels_like": {"type": "number"},
      "humidity": {"type": "number"},
      "description": {"type": "string"}
    }
  },
  "credit_cost": 1
}
```

### 3. Communication Tools

#### SendGrid Email

```json
{
  "name": "sendgrid_email",
  "display_name": "Send Email (SendGrid)",
  "description": "Send email using SendGrid API",
  "category": "communication",
  "method": "POST",
  "base_url": "https://api.sendgrid.com/v3/mail/send",
  "endpoint_path": "",
  "headers": {
    "Content-Type": "application/json"
  },
  "auth_type": "bearer",
  "auth_config": {
    "header_name": "Authorization",
    "prefix": "Bearer"
  },
  "input_schema": {
    "type": "object",
    "required": ["to", "subject", "body"],
    "properties": {
      "to": {
        "type": "string",
        "format": "email"
      },
      "subject": {"type": "string"},
      "body": {"type": "string"},
      "from": {
        "type": "string",
        "format": "email"
      }
    }
  },
  "output_schema": {
    "type": "object",
    "properties": {
      "message_id": {"type": "string"},
      "status": {"type": "string"}
    }
  },
  "credit_cost": 3
}
```

#### Twilio SMS

```json
{
  "name": "twilio_sms",
  "display_name": "Send SMS (Twilio)",
  "description": "Send SMS using Twilio API",
  "category": "communication",
  "method": "POST",
  "base_url": "https://api.twilio.com/2010-04-01",
  "endpoint_path": "/Accounts/{account_sid}/Messages.json",
  "auth_type": "basic",
  "auth_config": {
    "username_field": "account_sid",
    "password_field": "auth_token"
  },
  "input_schema": {
    "type": "object",
    "required": ["to", "body"],
    "properties": {
      "to": {
        "type": "string",
        "pattern": "^\\+[1-9]\\d{1,14}$"
      },
      "body": {
        "type": "string",
        "maxLength": 1600
      },
      "from": {"type": "string"}
    }
  },
  "credit_cost": 5
}
```

### 4. Slack Integration

```json
{
  "name": "slack_post_message",
  "display_name": "Post Slack Message",
  "description": "Post message to Slack channel",
  "category": "communication",
  "method": "POST",
  "base_url": "https://slack.com/api/chat.postMessage",
  "endpoint_path": "",
  "headers": {
    "Content-Type": "application/json"
  },
  "auth_type": "bearer",
  "auth_config": {
    "header_name": "Authorization",
    "prefix": "Bearer"
  },
  "input_schema": {
    "type": "object",
    "required": ["channel", "text"],
    "properties": {
      "channel": {"type": "string"},
      "text": {"type": "string"},
      "thread_ts": {"type": "string"}
    }
  },
  "credit_cost": 2
}
```

---

## 🔐 Authentication Types

### 1. API Key

```go
type APIKeyAuth struct {
    ParamName     string `json:"param_name"`     // e.g., "api_key", "appid"
    ParamLocation string `json:"param_location"` // "query", "header", "body"
    HeaderName    string `json:"header_name"`    // if location=header, e.g., "X-API-Key"
}
```

### 2. Bearer Token

```go
type BearerAuth struct {
    HeaderName string `json:"header_name"` // usually "Authorization"
    Prefix     string `json:"prefix"`      // usually "Bearer"
}
```

### 3. Basic Auth

```go
type BasicAuth struct {
    UsernameField string `json:"username_field"`
    PasswordField string `json:"password_field"`
}
```

### 4. OAuth 2.0

```go
type OAuth2Config struct {
    ClientID     string   `json:"client_id"`
    ClientSecret string   `json:"client_secret"`
    AuthURL      string   `json:"auth_url"`
    TokenURL     string   `json:"token_url"`
    Scopes       []string `json:"scopes"`
    RedirectURL  string   `json:"redirect_url"`
}
```

---

## 📡 API Endpoints

### Tool Management

```
GET    /api/v1/tools              # List available tools
POST   /api/v1/tools              # Create new tool (admin)
GET    /api/v1/tools/:id          # Get tool details
PUT    /api/v1/tools/:id          # Update tool (admin)
DELETE /api/v1/tools/:id          # Delete tool (admin)
GET    /api/v1/tools/:id/schema   # Get tool schema
```

### Tool Execution

```
POST   /api/v1/tools/:id/execute  # Execute tool
POST   /api/v1/tools/batch        # Execute multiple tools
```

### Tenant Configuration

```
GET    /api/v1/tenants/tools                    # List tools for tenant
POST   /api/v1/tenants/tools/:tool_id/configure # Configure tool for tenant
PUT    /api/v1/tenants/tools/:tool_id           # Update tenant tool config
DELETE /api/v1/tenants/tools/:tool_id           # Disable tool for tenant
```

### Analytics

```
GET    /api/v1/analytics/executions      # Get execution history
GET    /api/v1/analytics/usage           # Get usage statistics
GET    /api/v1/analytics/costs           # Get cost breakdown
```

---

## 🔄 Integration Points

### 1. Billing Service

```go
// Before tool execution
creditsRequired := tool.CreditCost
hasCredits, err := billingClient.CheckCredits(tenantID, creditsRequired)
if !hasCredits {
    return ErrInsufficientCredits
}

// After successful execution
err := billingClient.ConsumeCredits(tenantID, creditsRequired, toolID, executionID)
```

### 2. Kafka Events

```go
// Publish execution event
event := ToolExecutionEvent{
    ExecutionID:      uuid.New(),
    TenantID:         tenantID,
    ToolID:           toolID,
    Status:           "success",
    LatencyMS:        245,
    CreditsConsumed:  tool.CreditCost,
    Timestamp:        time.Now(),
}
kafka.Publish("tools.executions", event)
```

### 3. Auth Gateway

```go
// JWT validation via middleware
import "github.com/serphona/libs/platform-auth/middleware"

router.Use(middleware.RequireAuth())
```

### 4. Tenant Manager

```go
// Check tenant permissions
allowed, err := tenantClient.CanUseTool(tenantID, toolID)
```

---

## 🛡️ Security Considerations

### 1. Credential Storage

- ✅ Encrypt API keys and tokens in database
- ✅ Use separate encryption key per tenant
- ✅ Never log sensitive credentials
- ✅ Rotate credentials periodically

### 2. Input Validation

- ✅ Validate against JSON schemas
- ✅ Sanitize SQL injection attempts
- ✅ Prevent SSRF attacks
- ✅ Rate limit per tenant and per tool

### 3. Output Sanitization

- ✅ Remove sensitive data from responses
- ✅ Limit response size
- ✅ Validate output schema

### 4. Network Security

- ✅ Use HTTPS for all external calls
- ✅ Implement circuit breakers
- ✅ Set proper timeouts
- ✅ Whitelist allowed domains

---

## 📊 Observability

### Metrics

```go
// Prometheus metrics
tool_executions_total{tool_id, status}
tool_execution_duration_seconds{tool_id}
tool_rate_limit_hits_total{tenant_id, tool_id}
tool_credits_consumed_total{tenant_id, tool_id}
active_tool_executions{tool_id}
```

### Logging

```json
{
  "timestamp": "2025-12-10T14:20:00Z",
  "level": "info",
  "service": "tools-gateway",
  "event": "tool_execution",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "tool_id": "google_search",
  "execution_id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "success",
  "latency_ms": 245,
  "credits_consumed": 2
}
```

### Tracing

```go
// OpenTelemetry spans
span := tracer.StartSpan("execute_tool")
defer span.End()

span.SetAttributes(
    attribute.String("tool.id", toolID),
    attribute.String("tenant.id", tenantID),
)
```

---

## 🚀 Implementation Roadmap

### Phase 1: Core Infrastructure (Week 1-2)

- [x] Project structure setup
- [ ] Database schema and migrations
- [ ] Tool registry CRUD operations
- [ ] Input/output schema validation
- [ ] Basic HTTP client with retry logic
- [ ] Unit tests

### Phase 2: Authentication & Security (Week 3)

- [ ] API key authentication
- [ ] Bearer token authentication
- [ ] Basic auth support
- [ ] Credential encryption
- [ ] JWT middleware integration
- [ ] Security tests

### Phase 3: Built-in Tools (Week 4)

- [ ] Google Search integration
- [ ] OpenWeather integration
- [ ] SendGrid email integration
- [ ] Integration tests
- [ ] Tool documentation

### Phase 4: Rate Limiting & Billing (Week 5)

- [ ] Redis-based rate limiting
- [ ] Billing service integration
- [ ] Credit consumption logic
- [ ] Quota management
- [ ] Load tests

### Phase 5: Analytics & Observability (Week 6)

- [ ] Kafka event publishing
- [ ] Execution logging
- [ ] Prometheus metrics
- [ ] OpenTelemetry tracing
- [ ] Grafana dashboards

### Phase 6: Advanced Features (Week 7-8)

- [ ] OAuth 2.0 support
- [ ] Twilio SMS integration
- [ ] Slack integration
- [ ] Batch execution
- [ ] Webhook support
- [ ] Tool marketplace

### Phase 7: Production Ready (Week 9-10)

- [ ] Circuit breakers
- [ ] Health checks
- [ ] Docker containerization
- [ ] Kubernetes manifests
- [ ] CI/CD pipeline
- [ ] Documentation
- [ ] Performance optimization

---

## 📚 Technology Stack

### Core

- **Language**: Go 1.24+
- **Framework**: Gin
- **Database**: PostgreSQL 14+
- **Cache/Rate Limit**: Redis 7+
- **Messaging**: Kafka
- **HTTP Client**: `net/http` with `golang.org/x/net/context`

### Libraries

- **Validation**: `github.com/go-playground/validator`
- **JSON Schema**: `github.com/xeipuuv/gojsonschema`
- **JWT**: `github.com/golang-jwt/jwt`
- **Database**: `github.com/lib/pq`, `gorm.io/gorm`
- **Redis**: `github.com/redis/go-redis`
- **Kafka**: `github.com/segmentio/kafka-go`
- **Retry**: `github.com/avast/retry-go`
- **Circuit Breaker**: `github.com/sony/gobreaker`

### Observability

- **Logging**: `go.uber.org/zap`
- **Metrics**: `github.com/prometheus/client_golang`
- **Tracing**: `go.opentelemetry.io/otel`

---

## 🎯 Success Criteria

### Functional

- ✅ Execute tools with < 500ms overhead
- ✅ Support at least 10 different tool types
- ✅ 99.9% uptime
- ✅ Handle 1000 req/s per instance
- ✅ Zero credential leaks

### Non-Functional

- ✅ Comprehensive test coverage (>80%)
- ✅ Complete API documentation
- ✅ Monitoring dashboards
- ✅ Security audit passed
- ✅ Performance benchmarks documented

---

## 📝 Next Steps

1. **Review and approve** this planning document
2. **Setup development environment**
3. **Create database migrations**
4. **Implement Phase 1** (Core Infrastructure)
5. **Weekly progress reviews**

---

**Document Version**: 1.0  
**Created**: 10/12/2025  
**Last Updated**: 10/12/2025  
**Status**: 📋 Planning
