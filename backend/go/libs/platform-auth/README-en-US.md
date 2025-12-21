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

## Contributing

- Create a branch
- Make your changes
- Add tests
- Open a Pull Request

Additional docs: `IMPLEMENTATION_GUIDE-en-US.md` and `IMPLEMENTATION_GUIDE-pt-BR.md`.

---

Version: 1.0.0  
License: Proprietary
