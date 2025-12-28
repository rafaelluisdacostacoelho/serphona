# Platform Auth Library

Shared authentication utilities for Serphona microservices (the auth-gateway service keeps the full auth logic).

## Purpose

This library does **not** implement:
- Login/Logout
- User registration or database management
- OAuth providers
- Token issuance

This library **provides**:
- JWT validation middleware (Gin)
- HTTP client to talk to auth-gateway
- Shared types (Claims, User, tokens)
- JWT helpers
- Standardized errors

## Installation

```bash
go get github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth
```

## Usage

### 1. Authentication Middleware

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

func main() {
    router := gin.Default()

    // Public routes
    router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

    // Protected routes
    protected := router.Group("/api/v1")
    protected.Use(middleware.RequireAuth())
    {
        protected.GET("/billing/invoices", getInvoices)
        protected.GET("/tenants/current", getCurrentTenant)
    }

    router.Run(":8081")
}
```

### 2. Extract User Information

```go
func getInvoices(c *gin.Context) {
    claims, err := middleware.GetClaimsFromContext(c)
    if err != nil {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }

    invoices := getInvoicesForTenant(claims.TenantID)
    c.JSON(200, invoices)
}
```

### 3. HTTP Client for Auth Gateway

```go
import "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"

func main() {
    authClient := client.New("http://auth-gateway:8080")

    claims, err := authClient.ValidateToken(token)
    user, err := authClient.GetUserByID(userID, token)
}
```

#### Service-to-service calls with static bearer

Internal calls can carry a service token (client credentials) without overwriting user tokens:

```go
svcToken := os.Getenv("SERVICE_AUTH_TOKEN")
authClient := client.NewWithOptions(
    "http://auth-gateway:8080",
    client.WithStaticBearerToken(svcToken),
    client.WithServiceIdentity("billing-service", "billing-service-1"),
)

// If handler does not set Authorization, the static bearer is applied; existing user tokens stay untouched.
resp, err := authClient.ValidateToken("user-token") // keeps user token header
```

### 4. Manual JWT Validation

```go
import authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"

claims, err := authjwt.ValidateToken(tokenString)

// Extract token from Authorization header
rawToken, err := authjwt.ExtractTokenFromHeader(authHeader)
```

## Structure

```
platform-auth/
├── middleware/          # RequireAuth and role middlewares
├── client/              # HTTP client for auth-gateway
├── jwt/                 # JWT helpers (validation/extraction)
├── types/               # Claims, User and token types
├── errors/              # Standardized errors and codes
├── examples/            # Basic usage example with Gin
├── README*.md
├── IMPLEMENTATION_GUIDE*.md
├── go.mod
└── go.sum
```

## Configuration

Environment variables expected by consumers:

```env
JWT_SECRET=your-super-secret-jwt-key-change-in-production
AUTH_GATEWAY_URL=http://auth-gateway:8080
TENANT_ID_HEADER=X-Tenant-Id               # optional override; default matches library constant
TRACE_REQUEST_HEADER=X-Request-Id          # optional override in services
TLS_CA=/etc/ssl/certs/ca.pem               # optional client CA for auth-gateway calls
TLS_CERT=/etc/ssl/certs/client.pem         # optional client cert for mTLS
TLS_KEY=/etc/ssl/private/client.key        # optional client key for mTLS
```

Set the secret once during startup (or fail fast if missing):

```go
authjwt.MustSetSecretFromEnv()
```

## API Reference

- `middleware.RequireAuth()` — validates JWT and injects claims into context
- `middleware.RequireRole(role)` — enforces a specific role
- `middleware.RequireAdmin()` / `RequireSuperAdmin()` — convenience role guards
- `middleware.GetClaimsFromContext(c)` — returns `*types.Claims`
- `client.New(baseURL)` — creates auth-gateway client
- `client.ValidateToken(token)` — validates token via gateway
- `client.GetUserByID(userID, token)` / `client.GetMe(token)`
- `client.RefreshToken(refreshToken)` / `client.Logout(token)`
- `jwt.ValidateToken(token)` — local validation using configured secret
- `jwt.ExtractTokenFromHeader(header)` — parses `Authorization: Bearer <token>`
- `middleware.TenantIDFromContext(ctx)` — resolves tenant from claims or explicit context
- `middleware.EnsureTenantHeader(headers, tenantID)` — injects tenant id on outbound requests without mutating caller headers

## Security

- Validate tokens locally for regular requests; call auth-gateway for sensitive flows.
- Keep `JWT_SECRET` consistent across services and never log it.
- Always use HTTPS in production.

## Tests

```bash
go test ./...
```

## Examples

See `examples/` for a runnable Gin example.

### Tenant/RLS helper usage (DB, Kafka, outbound HTTP)

```go
// Resolve tenant from claims and enforce before DB/Kafka operations
tenantID, err := middleware.TenantIDFromContext(ctx)
if err != nil {
    return err // missing tenant
}
if err := middleware.EnforceTenant(ctx, tenantIDFromPayload); err != nil {
    return err // cross-tenant guard
}

// Propagate tenant to Kafka headers or outbound HTTP
headers := middleware.EnsureTenantHeader(nil, tenantID)
kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{Key: middleware.TenantIDHeader, Value: []byte(tenantID)})
req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
req.Header = middleware.EnsureTenantHeader(req.Header, tenantID)
```

The helpers prefer the tenant present in JWT claims; `WithTenantID` can be used to set a tenant explicitly for system-to-system flows.

## Contributing

- Create a branch
- Make your changes
- Add tests
- Open a Pull Request

Additional docs: `IMPLEMENTATION_GUIDE-en-US.md` and `IMPLEMENTATION_GUIDE-pt-BR.md`.

---

Version: 1.0.0  
License: Proprietary
