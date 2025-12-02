# Libs vs Services - Serphona Architecture Guide

> 📋 Complete guide on when to use Libs (shared libraries) vs Services (microservices) in the Serphona project.

## 📌 Overview

In Serphona, we follow a microservices architecture where:
- **Services** are independent and autonomous applications
- **Libs** are shared libraries between services

---

## 🚀 SERVICES (Microservices)

### What are they?

Services are **complete and independent applications** that implement a specific business domain.

### Service Characteristics:

✅ **Own HTTP/gRPC server** (run on different ports)  
✅ **Own database** (or isolated schema)  
✅ **Complete business logic** for their domain  
✅ **Public REST/gRPC APIs**  
✅ **Independent deployment** (Docker containers, Kubernetes pods)  
✅ **Independent scalability** (can scale horizontally/vertically)  
✅ **Own lifecycle** (versions, releases, rollbacks)

### Services in Serphona:

```
backend/go/services/
├── auth-gateway/           → Authentication and authorization
├── billing-service/        → Payments, subscriptions, credit wallet
├── tenant-manager/         → Tenant/organization management
├── agent-orchestrator/     → AI agent orchestration
├── analytics-query-service/→ Analytics queries and reports
└── tools-gateway/          → Gateway for external tools
```

### Typical Service Structure:

```
auth-gateway/
├── cmd/
│   └── server/
│       └── main.go              # Server entry point
├── internal/
│   ├── domain/                  # Entities and business rules
│   ├── usecase/                 # Use cases
│   ├── service/                 # Domain services
│   ├── adapter/                 # Adapters (HTTP, DB, OAuth)
│   └── config/                  # Configuration
├── migrations/                  # Database migrations
├── go.mod                       # Go dependencies
├── Dockerfile                   # Docker container
├── .env.example                 # Environment variables
└── README.md                    # Documentation
```

### When to use a SERVICE:

- ✅ Needs **complete business logic** (e.g., entire authentication flow)
- ✅ Needs **own HTTP/gRPC API**
- ✅ Needs to **manage own state/data**
- ✅ Functionality should be **deployed independently**
- ✅ Needs to **scale independently** of other components
- ✅ Has **clear responsibility** over a business domain

### Practical Examples:

#### auth-gateway
```
Responsibilities:
• Login/Logout
• User registration
• OAuth (Google, Microsoft, Apple)
• JWT issuance and validation
• Session management
• Refresh tokens

Exposes API at: http://localhost:8080
```

#### billing-service
```
Responsibilities:
• Stripe integration
• Subscription management
• Credit wallet
• Credit top-up
• Credit consumption
• Payment webhooks

Exposes API at: http://localhost:8081
```

#### tenant-manager
```
Responsibilities:
• Tenant/organization CRUD
• Quota management
• Tenant configurations
• Members and permissions
• Multi-tenant isolation

Exposes API at: http://localhost:8082
```

---

## 📚 LIBS (Shared Libraries)

### What are they?

Libs are **reusable shared code** between multiple services, without their own business logic.

### Lib Characteristics:

❌ **No HTTP server**  
❌ **No own database**  
✅ **Imported** by other services via `go mod`  
✅ **Contain utilities, helpers, common interfaces**  
✅ **Shared middleware**  
✅ **HTTP clients** for inter-service communication  
✅ **Shared types and contracts**

### Libs in Serphona:

```
backend/go/libs/
├── platform-core/         → Common configurations, utilities
├── platform-events/       → Messaging/event system
├── platform-observability/→ Logging, metrics, tracing
└── platform-auth/         → Authentication middleware, JWT validation
```

### Typical Lib Structure:

```
platform-auth/
├── middleware/
│   ├── jwt_validator.go    # JWT validation middleware
│   └── auth.go             # Authentication middleware
├── client/
│   └── auth_client.go      # HTTP client for auth-gateway
├── types/
│   ├── claims.go           # JWT claims structure
│   └── user.go             # Shared user types
├── errors/
│   └── errors.go           # Standardized authentication errors
├── go.mod                  # Lib dependencies
└── README.md               # Lib documentation
```

### When to use a LIB:

- ✅ Code **reused by multiple services**
- ✅ **Utilities, helpers, constants** common
- ✅ **Shared middleware** (auth, logging, cors)
- ✅ **HTTP client** for inter-service communication
- ✅ **Event/message definitions** (pub/sub)
- ✅ **Shared types and interfaces**
- ✅ **Common configurations**

### Practical Examples:

#### platform-auth
```go
// Authentication middleware used by all services
import "github.com/serphona/backend/go/libs/platform-auth/middleware"

func setupRouter() *gin.Engine {
    router := gin.Default()
    
    // Protected routes
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
// Event system used by all services
import "github.com/serphona/backend/go/libs/platform-events"

// Publish event
err := events.Publish("user.created", UserCreatedEvent{
    UserID:   user.ID,
    TenantID: user.TenantID,
})

// Subscribe to event
events.Subscribe("user.created", func(event UserCreatedEvent) {
    // Create resources for new user
    createUserResources(event.UserID)
})
```

#### platform-core
```go
// Common configurations
import "github.com/serphona/backend/go/libs/platform-core/config"

// Shared logger
import "github.com/serphona/backend/go/libs/platform-core/logger"

func main() {
    cfg := config.Load()
    log := logger.New(cfg.LogLevel)
    
    log.Info("Starting service", "name", cfg.ServiceName)
}
```

#### platform-observability
```go
// Metrics and tracing
import "github.com/serphona/backend/go/libs/platform-observability/metrics"
import "github.com/serphona/backend/go/libs/platform-observability/tracing"

// Record metric
metrics.RecordLatency("api.request", duration)

// Create tracing span
span := tracing.StartSpan("process_payment")
defer span.End()
```

---

## 🏗️ Complete Serphona Architecture

### Component Diagram:

```
┌─────────────────────────────────────────────────────────┐
│                     API Gateway                          │
│               (Kong, Traefik, or NGINX)                  │
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
        │     Shared Libs (Go Modules)         │
        ├─────────────────────────────────────┤
        │ • platform-auth                      │
        │ • platform-core                      │
        │ • platform-events                    │
        │ • platform-observability             │
        └──────────────────────────────────────┘
```

### Inter-Service Communication:

```go
// billing-service needs to validate authenticated user
// Uses middleware from platform-auth lib

import "github.com/serphona/libs/platform-auth/middleware"

router.Use(middleware.RequireAuth())

// The middleware:
// 1. Extracts JWT from Authorization header
// 2. Validates token signature
// 3. Extracts claims (userID, tenantID, roles)
// 4. Injects into request context
```

### Complete Request Flow:

```
1. Client → API Gateway
   POST /api/v1/billing/subscribe
   Authorization: Bearer eyJhbGc...

2. API Gateway → billing-service (port 8081)
   Routes to correct service

3. billing-service → platform-auth middleware
   Validates JWT locally (without calling auth-gateway)

4. billing-service → business logic
   Creates subscription in Stripe

5. billing-service → platform-events
   Publishes "subscription.created" event

6. tenant-manager → listens to event
   Updates tenant quota

7. billing-service → Response to client
   Returns created subscription data
```

---

## 📊 Comparison: Libs vs Services

| Aspect | Services | Libs |
|---------|----------|------|
| **Purpose** | Complete business logic | Shared/utility code |
| **Deployment** | Independent (own container) | Included in services that use it |
| **HTTP Server** | ✅ Yes, own | ❌ No |
| **Database** | ✅ Yes, own | ❌ No |
| **Public API** | ✅ Yes, REST/gRPC | ❌ No |
| **Scalability** | ✅ Independent | 📦 Scales with service |
| **Versioning** | ✅ Independent releases | 📦 Via go.mod in services |
| **Examples** | auth-gateway, billing-service | platform-auth, platform-events |

---

## 🎯 Architecture Decision: Auth

### ❓ Original Question:

> "I have folders `go/libs/platform-auth` and `services/auth-gateway`, which should I use for login, register, etc?"

### ✅ Answer:

**Use `services/auth-gateway` for login, register, OAuth, etc.**

**Reason:**
- Authentication is **complex business logic**
- Needs **database** (users, sessions, oauth_states)
- Needs **HTTP API** for frontend/mobile consumption
- Needs to manage **state** (sessions, tokens)
- Needs to **integrate** with OAuth providers (Google, Microsoft, Apple)

**Current situation:**
- ✅ `services/auth-gateway`: Complete and functional service
- ❌ `libs/platform-auth`: Only go.mod, no implementation

### 🔄 Recommended Refactoring:

**`services/auth-gateway`** (keep as is):
```
Responsibilities:
✅ Login/Logout
✅ User registration
✅ OAuth providers
✅ JWT issuance
✅ Session management
✅ User database
```

**`libs/platform-auth`** (refactor to contain):
```
Responsibilities:
✅ JWT validation middleware (used by other services)
✅ HTTP client to call auth-gateway
✅ Shared types (Claims, User, etc)
✅ Standardized authentication errors
```

---

## 📝 Usage Patterns

### Pattern 1: Service exposes API, Lib provides client

#### Service (auth-gateway):
```go
// auth-gateway/internal/adapter/http/handler/auth_handler.go

func (h *AuthHandler) Login(c *gin.Context) {
    // Complete login logic
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
    // Calls auth-gateway to validate token
    resp, err := http.Get(c.baseURL + "/api/v1/auth/validate")
    // ...
}
```

#### Other services use the lib:
```go
// billing-service/main.go

import "github.com/serphona/libs/platform-auth/client"

authClient := client.NewAuthClient("http://auth-gateway:8080")
claims, err := authClient.ValidateToken(token)
```

### Pattern 2: Lib provides shared middleware

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

#### All services use the middleware:
```go
// billing-service/main.go
// tenant-manager/main.go
// agent-orchestrator/main.go

import "github.com/serphona/libs/platform-auth/middleware"

router.Use(middleware.RequireAuth())
```

### Pattern 3: Lib provides event system

#### Lib (platform-events):
```go
// platform-events/publisher.go

func Publish(topic string, data interface{}) error {
    // Publishes to RabbitMQ/Redis/Kafka
}

func Subscribe(topic string, handler func(interface{})) error {
    // Subscribes to topic
}
```

#### Services publish and consume events:
```go
// auth-gateway publishes
events.Publish("user.created", UserCreatedEvent{...})

// billing-service consumes
events.Subscribe("user.created", func(event UserCreatedEvent) {
    createFreeTrialSubscription(event.UserID)
})

// tenant-manager consumes
events.Subscribe("user.created", func(event UserCreatedEvent) {
    incrementTenantUserCount(event.TenantID)
})
```

---

## 🔧 Dependencies between Services and Libs

### Services depend on Libs:

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

### Libs do NOT depend on Services:

```go
// platform-auth/go.mod

module github.com/serphona/backend/go/libs/platform-auth

require (
    github.com/golang-jwt/jwt/v5 v5.1.0
    // Should NOT have: github.com/serphona/.../auth-gateway
)
```

### Libs can depend on other Libs:

```go
// platform-events/go.mod

require (
    github.com/serphona/backend/go/libs/platform-core v1.2.0
    github.com/serphona/backend/go/libs/platform-observability v1.0.0
)
```

---

## 🚀 Best Practices

### Services:

1. ✅ **Keep services focused** on a specific domain
2. ✅ **Use Clean Architecture** (domain, usecase, adapter)
3. ✅ **Expose well-documented APIs** (OpenAPI/Swagger)
4. ✅ **Implement health checks** (`/health`, `/ready`)
5. ✅ **Use migrations** to evolve the database
6. ✅ **Have detailed README** with setup instructions
7. ✅ **Configure observability** (logs, metrics, traces)
8. ✅ **Implement circuit breakers** for external dependencies

### Libs:

1. ✅ **Keep libs lightweight** and without heavy dependencies
2. ✅ **Document well** public functions
3. ✅ **Use interfaces** to facilitate testing
4. ✅ **Version appropriately** (semantic versioning)
5. ✅ **Avoid business logic** in libs
6. ✅ **Maintain backward compatibility** when possible
7. ✅ **Unit test** everything that is public
8. ✅ **Provide examples** of usage in README

---

## 📚 Additional Resources

### Related Documentation:

- [Auth Gateway README](../backend/go/services/auth-gateway/README.md)
- [Billing Service Prompts](../backend/go/services/billing-service/prompts/)
- [Tenant Manager Docs](../backend/go/services/tenant-manager/docs/)

### Architecture Patterns:

- Clean Architecture (Uncle Bob)
- Hexagonal Architecture (Ports & Adapters)
- Microservices Patterns (Chris Richardson)
- Domain-Driven Design (Eric Evans)

### Technologies Used:

- **Backend**: Go 1.21+
- **Database**: PostgreSQL 14+
- **Messaging**: RabbitMQ / Redis
- **API Gateway**: Kong / Traefik
- **Observability**: Prometheus, Grafana, Jaeger
- **Deployment**: Docker, Kubernetes

---

## ✅ Decision Checklist

When creating a new component, use this checklist:

### I should create a SERVICE when:

- [ ] I need to expose HTTP/gRPC API
- [ ] I need to manage persistent data
- [ ] I have complex business logic
- [ ] I need to scale independently
- [ ] I need independent deployment
- [ ] I have a clear bounded context

### I should create a LIB when:

- [ ] Code will be reused by 2+ services
- [ ] It's middleware or utility
- [ ] It's shared types/interfaces
- [ ] It's HTTP client for communication
- [ ] It's event/messaging system
- [ ] It's common configurations

---

## 🎯 Conclusion

Serphona's microservices architecture follows the principle of **clear separation of responsibilities**:

- **Services** implement **business logic** and expose **APIs**
- **Libs** provide **shared code** and **utilities**

This separation ensures:
- ✅ **Maintainability**: Each service has clear responsibility
- ✅ **Scalability**: Services can scale independently
- ✅ **Reusability**: Libs avoid code duplication
- ✅ **Testability**: Isolated components are easier to test
- ✅ **Independent deployment**: Services can be updated without affecting others

**Remember**: When in doubt, start with a **service**. It's easier to extract shared code into a lib later than to transform a lib into a service.

---

**Last updated**: 11/29/2025  
**Version**: 1.0  
**Author**: Serphona Team
