# Libs vs Services - Guia de Arquitetura Serphona

> 📋 Guia completo sobre quando usar Libs (bibliotecas compartilhadas) vs Services (microserviços) no projeto Serphona.

## 📌 Visão Geral

No Serphona, seguimos uma arquitetura de microserviços onde:
- **Services** são aplicações independentes e autônomas
- **Libs** são bibliotecas compartilhadas entre os services

---

## 🚀 SERVICES (Microserviços)

### O que são?

Services são **aplicações completas e independentes** que implementam um domínio de negócio específico.

### Características dos Services:

✅ **Servidor HTTP/gRPC próprio** (executam em portas diferentes)  
✅ **Banco de dados próprio** (ou schema isolado)  
✅ **Lógica de negócio completa** para seu domínio  
✅ **APIs REST/gRPC públicas**  
✅ **Deploy independente** (Docker containers, Kubernetes pods)  
✅ **Escalabilidade independente** (pode escalar horizontal/verticalmente)  
✅ **Ciclo de vida próprio** (versões, releases, rollbacks)

### Services no Serphona:

```
backend/go/services/
├── auth-gateway/           → Autenticação e autorização
├── billing-service/        → Pagamentos, assinaturas, wallet de créditos
├── tenant-manager/         → Gestão de tenants/organizações
├── agent-orchestrator/     → Orquestração de agentes AI
├── analytics-query-service/→ Consultas e relatórios de analytics
└── tools-gateway/          → Gateway para ferramentas externas
```

### Estrutura típica de um Service:

```
auth-gateway/
├── cmd/
│   └── server/
│       └── main.go              # Entry point do servidor
├── internal/
│   ├── domain/                  # Entidades e regras de negócio
│   ├── usecase/                 # Casos de uso
│   ├── service/                 # Serviços de domínio
│   ├── adapter/                 # Adapters (HTTP, DB, OAuth)
│   └── config/                  # Configuração
├── migrations/                  # Migrations de banco de dados
├── go.mod                       # Dependências Go
├── Dockerfile                   # Container Docker
├── .env.example                 # Variáveis de ambiente
└── README.md                    # Documentação
```

### Quando usar um SERVICE:

- ✅ Precisa de **lógica de negócio completa** (ex: todo o fluxo de autenticação)
- ✅ Precisa de **API HTTP/gRPC própria**
- ✅ Precisa **gerenciar estado/dados próprios**
- ✅ Funcionalidade deve ser **deployada independentemente**
- ✅ Precisa **escalar independentemente** de outros componentes
- ✅ Tem **responsabilidade clara** sobre um domínio de negócio

### Exemplos práticos:

#### auth-gateway
```
Responsabilidades:
• Login/Logout
• Registro de usuários
• OAuth (Google, Microsoft, Apple)
• Emissão e validação de JWT
• Gestão de sessões
• Refresh tokens

Expõe API em: http://localhost:8080
```

#### billing-service
```
Responsabilidades:
• Integração com Stripe
• Gestão de assinaturas
• Wallet de créditos
• Top-up de créditos
• Consumo de créditos
• Webhooks de pagamento

Expõe API em: http://localhost:8081
```

#### tenant-manager
```
Responsabilidades:
• CRUD de tenants/organizações
• Gestão de quotas
• Configurações de tenant
• Membros e permissões
• Isolamento multi-tenant

Expõe API em: http://localhost:8082
```

---

## 📚 LIBS (Bibliotecas Compartilhadas)

### O que são?

Libs são **código reutilizável compartilhado** entre múltiplos services, sem lógica de negócio própria.

### Características das Libs:

❌ **Não possuem servidor HTTP**  
❌ **Não possuem banco de dados próprio**  
✅ **São importadas** por outros services via `go mod`  
✅ **Contêm utilitários, helpers, interfaces comuns**  
✅ **Middleware compartilhado**  
✅ **Clientes HTTP** para comunicação entre services  
✅ **Tipos e contratos compartilhados**

### Libs no Serphona:

```
backend/go/libs/
├── platform-core/         → Configurações, utilitários comuns
├── platform-events/       → Sistema de mensageria/eventos
├── platform-observability/→ Logging, métricas, tracing
└── platform-auth/         → Middleware de autenticação, validação JWT
```

### Estrutura típica de uma Lib:

```
platform-auth/
├── middleware/
│   ├── jwt_validator.go    # Middleware de validação JWT
│   └── auth.go             # Middleware de autenticação
├── client/
│   └── auth_client.go      # Cliente HTTP para auth-gateway
├── types/
│   ├── claims.go           # Estrutura de claims JWT
│   └── user.go             # Tipos compartilhados de usuário
├── errors/
│   └── errors.go           # Erros de autenticação padronizados
├── go.mod                  # Dependências da lib
└── README.md               # Documentação da lib
```

### Quando usar uma LIB:

- ✅ Código **reutilizado por múltiplos services**
- ✅ **Utilitários, helpers, constantes** comuns
- ✅ **Middleware compartilhado** (auth, logging, cors)
- ✅ **Cliente HTTP** para comunicação entre services
- ✅ **Definições de eventos/mensagens** (pub/sub)
- ✅ **Tipos e interfaces** compartilhados
- ✅ **Configurações** comuns

### Exemplos práticos:

#### platform-auth
```go
// Middleware de autenticação usado por todos os services
import "github.com/serphona/backend/go/libs/platform-auth/middleware"

func setupRouter() *gin.Engine {
    router := gin.Default()
    
    // Rotas protegidas
    protected := router.Group("/api/v1")
    protected.Use(middleware.RequireAuth())
    {
        protected.GET("/billing/invoices", getInvoices)
        protected.GET("/tenants/current", getCurrentTenant)
    }
    
    return router
}
```

#### platform-events
```go
// Sistema de eventos usado por todos os services
import "github.com/serphona/backend/go/libs/platform-events"

// Publicar evento
err := events.Publish("user.created", UserCreatedEvent{
    UserID:   user.ID,
    TenantID: user.TenantID,
})

// Assinar evento
events.Subscribe("user.created", func(event UserCreatedEvent) {
    // Criar recursos para novo usuário
    createUserResources(event.UserID)
})
```

#### platform-core
```go
// Configurações comuns
import "github.com/serphona/backend/go/libs/platform-core/config"

// Logger compartilhado
import "github.com/serphona/backend/go/libs/platform-core/logger"

func main() {
    cfg := config.Load()
    log := logger.New(cfg.LogLevel)
    
    log.Info("Starting service", "name", cfg.ServiceName)
}
```

#### platform-observability
```go
// Métricas e tracing
import "github.com/serphona/backend/go/libs/platform-observability/metrics"
import "github.com/serphona/backend/go/libs/platform-observability/tracing"

// Registrar métrica
metrics.RecordLatency("api.request", duration)

// Criar span de tracing
span := tracing.StartSpan("process_payment")
defer span.End()
```

---

## 🏗️ Arquitetura Completa do Serphona

### Diagrama de Componentes:

```
┌─────────────────────────────────────────────────────────┐
│                     API Gateway                          │
│               (Kong, Traefik, ou NGINX)                  │
│                   Port: 80/443                           │
└─────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
┌───────▼─────────┐ ┌─────▼──────────┐ ┌────▼──────────┐
│ auth-gateway    │ │ billing-service│ │ tenant-manager│
│   Port: 8080    │ │   Port: 8081   │ │  Port: 8082   │
│                 │ │                │ │               │
│ • Login         │ │ • Stripe       │ │ • Tenants     │
│ • Register      │ │ • Wallet       │ │ • Quotas      │
│ • OAuth         │ │ • Subscriptions│ │ • Members     │
└─────────────────┘ └────────────────┘ └───────────────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
        ┌──────────────────┴──────────────────┐
        │     Libs Compartilhadas (Go Modules) │
        ├─────────────────────────────────────┤
        │ • platform-auth                      │
        │ • platform-core                      │
        │ • platform-events                    │
        │ • platform-observability             │
        └──────────────────────────────────────┘
```

### Comunicação entre Services:

```go
// billing-service precisa validar usuário autenticado
// Usa middleware da lib platform-auth

import "github.com/serphona/libs/platform-auth/middleware"

router.Use(middleware.RequireAuth())

// O middleware:
// 1. Extrai JWT do header Authorization
// 2. Valida assinatura do token
// 3. Extrai claims (userID, tenantID, roles)
// 4. Injeta no context da request
```

### Fluxo de Request Completo:

```
1. Cliente → API Gateway
   POST /api/v1/billing/subscribe
   Authorization: Bearer eyJhbGc...

2. API Gateway → billing-service (port 8081)
   Roteia para o service correto

3. billing-service → platform-auth middleware
   Valida JWT localmente (sem chamar auth-gateway)

4. billing-service → lógica de negócio
   Cria assinatura no Stripe

5. billing-service → platform-events
   Publica evento "subscription.created"

6. tenant-manager → escuta evento
   Atualiza quota do tenant

7. billing-service → Response ao cliente
   Retorna dados da assinatura criada
```

---

## 📊 Comparação: Libs vs Services

| Aspecto | Services | Libs |
|---------|----------|------|
| **Propósito** | Lógica de negócio completa | Código compartilhado/utilitário |
| **Deploy** | Independente (container próprio) | Incluída nos services que a usam |
| **Servidor HTTP** | ✅ Sim, próprio | ❌ Não |
| **Banco de dados** | ✅ Sim, próprio | ❌ Não |
| **API pública** | ✅ Sim, REST/gRPC | ❌ Não |
| **Escalabilidade** | ✅ Independente | 📦 Escala com o service |
| **Versionamento** | ✅ Releases independentes | 📦 Via go.mod nos services |
| **Exemplos** | auth-gateway, billing-service | platform-auth, platform-events |

---

## 🎯 Decisão de Arquitetura: Auth

### ❓ Pergunta Original:

> "Eu tenho as pastas `go/libs/platform-auth` e `services/auth-gateway`, qual devo usar para login, register, etc?"

### ✅ Resposta:

**Use `services/auth-gateway` para login, register, OAuth, etc.**

**Motivo:**
- Autenticação é **lógica de negócio complexa**
- Precisa de **banco de dados** (users, sessions, oauth_states)
- Precisa de **API HTTP** para frontend/mobile consumir
- Precisa gerenciar **estado** (sessões, tokens)
- Precisa **integrar** com providers OAuth (Google, Microsoft, Apple)

**Situação atual:**
- ✅ `services/auth-gateway`: Serviço completo e funcional
- ❌ `libs/platform-auth`: Apenas go.mod, sem implementação

### 🔄 Refatoração Recomendada:

**`services/auth-gateway`** (mantém como está):
```
Responsabilidades:
✅ Login/Logout
✅ Registro de usuários
✅ OAuth providers
✅ Emissão de JWT
✅ Gestão de sessões
✅ Database de users
```

**`libs/platform-auth`** (refatorar para conter):
```
Responsabilidades:
✅ Middleware JWT validation (usado por outros services)
✅ Cliente HTTP para chamar auth-gateway
✅ Tipos compartilhados (Claims, User, etc)
✅ Erros de autenticação padronizados
```

---

## 📝 Padrões de Uso

### Padrão 1: Service expõe API, Lib fornece cliente

#### Service (auth-gateway):
```go
// auth-gateway/internal/adapter/http/handler/auth_handler.go

func (h *AuthHandler) Login(c *gin.Context) {
    // Lógica completa de login
    user, tokens, err := h.authUseCase.Login(req.Email, req.Password)
    
    c.JSON(200, gin.H{
        "user": user,
        "tokens": tokens,
    })
}
```

#### Lib (platform-auth):
```go
// platform-auth/client/auth_client.go

type AuthClient struct {
    baseURL string
}

func (c *AuthClient) ValidateToken(token string) (*Claims, error) {
    // Chama auth-gateway para validar token
    resp, err := http.Get(c.baseURL + "/api/v1/auth/validate")
    // ...
}
```

#### Outros services usam a lib:
```go
// billing-service/main.go

import "github.com/serphona/libs/platform-auth/client"

authClient := client.NewAuthClient("http://auth-gateway:8080")
claims, err := authClient.ValidateToken(token)
```

### Padrão 2: Lib fornece middleware compartilhado

#### Lib (platform-auth):
```go
// platform-auth/middleware/jwt.go

func RequireAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        
        claims, err := jwt.ValidateToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
            return
        }
        
        c.Set("userID", claims.UserID)
        c.Set("tenantID", claims.TenantID)
        c.Next()
    }
}
```

#### Todos os services usam o middleware:
```go
// billing-service/main.go
// tenant-manager/main.go
// agent-orchestrator/main.go

import "github.com/serphona/libs/platform-auth/middleware"

router.Use(middleware.RequireAuth())
```

### Padrão 3: Lib fornece sistema de eventos

#### Lib (platform-events):
```go
// platform-events/publisher.go

func Publish(topic string, data interface{}) error {
    // Publica no RabbitMQ/Redis/Kafka
}

func Subscribe(topic string, handler func(interface{})) error {
    // Assina tópico
}
```

#### Services publicam e consomem eventos:
```go
// auth-gateway publica
events.Publish("user.created", UserCreatedEvent{...})

// billing-service consome
events.Subscribe("user.created", func(event UserCreatedEvent) {
    createFreeTrialSubscription(event.UserID)
})

// tenant-manager consome
events.Subscribe("user.created", func(event UserCreatedEvent) {
    incrementTenantUserCount(event.TenantID)
})
```

---

## 🔧 Dependências entre Services e Libs

### Services dependem de Libs:

```go
// billing-service/go.mod

module github.com/serphona/backend/go/services/billing-service

require (
    github.com/serphona/backend/go/libs/platform-auth v1.0.0
    github.com/serphona/backend/go/libs/platform-core v1.2.0
    github.com/serphona/backend/go/libs/platform-events v1.1.0
    github.com/serphona/backend/go/libs/platform-observability v1.0.0
)
```

### Libs NÃO dependem de Services:

```go
// platform-auth/go.mod

module github.com/serphona/backend/go/libs/platform-auth

require (
    github.com/golang-jwt/jwt/v5 v5.1.0
    // NÃO deve ter: github.com/serphona/.../auth-gateway
)
```

### Libs podem depender de outras Libs:

```go
// platform-events/go.mod

require (
    github.com/serphona/backend/go/libs/platform-core v1.2.0
    github.com/serphona/backend/go/libs/platform-observability v1.0.0
)
```

---

## 🚀 Melhores Práticas

### Services:

1. ✅ **Mantenha services focados** em um domínio específico
2. ✅ **Use Clean Architecture** (domain, usecase, adapter)
3. ✅ **Exponha APIs bem documentadas** (OpenAPI/Swagger)
4. ✅ **Implemente health checks** (`/health`, `/ready`)
5. ✅ **Use migrations** para evoluir o banco de dados
6. ✅ **Tenha README detalhado** com instruções de setup
7. ✅ **Configure observabilidade** (logs, métricas, traces)
8. ✅ **Implemente circuit breakers** para dependências externas

### Libs:

1. ✅ **Mantenha libs leves** e sem dependências pesadas
2. ✅ **Documente bem** as funções públicas
3. ✅ **Use interfaces** para facilitar testes
4. ✅ **Versione adequadamente** (semantic versioning)
5. ✅ **Evite lógica de negócio** em libs
6. ✅ **Mantenha retrocompatibilidade** quando possível
7. ✅ **Teste unitariamente** tudo que é público
8. ✅ **Forneça exemplos** de uso no README

---

## 📚 Recursos Adicionais

### Documentação Relacionada:

- [Auth Gateway README](../backend/go/services/auth-gateway/README.md)
- [Billing Service Prompts](../backend/go/services/billing-service/prompts/)
- [Tenant Manager Docs](../backend/go/services/tenant-manager/docs/)

### Padrões de Arquitetura:

- Clean Architecture (Uncle Bob)
- Hexagonal Architecture (Ports & Adapters)
- Microservices Patterns (Chris Richardson)
- Domain-Driven Design (Eric Evans)

### Tecnologias Usadas:

- **Backend**: Go 1.21+
- **Database**: PostgreSQL 14+
- **Messaging**: RabbitMQ / Redis
- **API Gateway**: Kong / Traefik
- **Observability**: Prometheus, Grafana, Jaeger
- **Deployment**: Docker, Kubernetes

---

## ✅ Checklist de Decisão

Ao criar um novo componente, use este checklist:

### Devo criar um SERVICE quando:

- [ ] Preciso expor API HTTP/gRPC
- [ ] Preciso gerenciar dados persistentes
- [ ] Tenho lógica de negócio complexa
- [ ] Preciso escalar independentemente
- [ ] Preciso deploy independente
- [ ] Tenho um bounded context claro

### Devo criar uma LIB quando:

- [ ] Código será reutilizado por 2+ services
- [ ] É middleware ou utilitário
- [ ] São tipos/interfaces compartilhados
- [ ] É cliente HTTP para comunicação
- [ ] É sistema de eventos/mensageria
- [ ] São configurações comuns

---

## 🎯 Conclusão

A arquitetura de microserviços do Serphona segue o princípio de **separação clara de responsabilidades**:

- **Services** implementam **lógica de negócio** e expõem **APIs**
- **Libs** fornecem **código compartilhado** e **utilitários**

Esta separação garante:
- ✅ **Manutenibilidade**: Cada service tem responsabilidade clara
- ✅ **Escalabilidade**: Services podem escalar independentemente
- ✅ **Reutilização**: Libs evitam duplicação de código
- ✅ **Testabilidade**: Componentes isolados são mais fáceis de testar
- ✅ **Deploy independente**: Services podem ser atualizados sem afetar outros

**Lembre-se**: Quando em dúvida, comece com um **service**. É mais fácil extrair código compartilhado para uma lib depois do que transformar uma lib em um service.

---

**Última atualização**: 29/11/2025  
**Versão**: 1.0  
**Autor**: Equipe Serphona
