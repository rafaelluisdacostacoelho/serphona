# Platform Auth Library

> 🔐 Shared authentication library for Serphona microservices.

## 📋 Purpose

This library provides reusable authentication components to be used by all Serphona services, **except** the `auth-gateway` which implements the complete authentication logic.

## 🎯 Responsibilities

`platform-auth` does **NOT** implement:
- ❌ Login/Logout
- ❌ User registration
- ❌ User database management
- ❌ OAuth providers
- ❌ Token issuance

`platform-auth` **PROVIDES**:
- ✅ JWT validation middleware
- ✅ HTTP client to call auth-gateway
- ✅ Shared types (Claims, User, etc)
- ✅ JWT utilities
- ✅ Standardized errors

## 📦 Installation

```bash
go get github.com/serphona/backend/go/libs/platform-auth
```

## 🚀 Usage

### 1. Authentication Middleware

Use in any service to protect routes:

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/serphona/backend/go/libs/platform-auth/middleware"
)

func main() {
    router := gin.Default()
    
    // Public routes
    router.GET("/health", healthHandler)
    
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
    // Extract claims from context (injected by middleware)
    userID := c.GetString("userID")
    tenantID := c.GetString("tenantID")
    role := c.GetString("role")
    
    // Or use the helper
    claims, err := middleware.GetClaimsFromContext(c)
    if err != nil {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }
    
    // Use the information
    invoices := getInvoicesForTenant(claims.TenantID)
    c.JSON(200, invoices)
}
```

### 3. HTTP Client for Auth Gateway

```go
package main

import (
    "github.com/serphona/backend/go/libs/platform-auth/client"
)

func main() {
    // Create client
    authClient := client.New("http://auth-gateway:8080")
    
    // Validate token
    claims, err := authClient.ValidateToken(token)
    if err != nil {
        // Invalid token
    }
    
    // Get user information
    user, err := authClient.GetUserByID(userID)
}
```

### 4. Manual JWT Validation

```go
import "github.com/serphona/backend/go/libs/platform-auth/jwt"

// Validate token manually
claims, err := jwt.ValidateToken(tokenString, jwtSecret)
if err != nil {
    // Invalid token
}

// Extract token from Authorization header
token, err := jwt.ExtractTokenFromHeader(authHeader)
```

## 📁 Structure

```
platform-auth/
├── middleware/
│   ├── auth.go           # RequireAuth() middleware
│   └── context.go        # Context helpers
├── client/
│   └── auth_client.go    # HTTP client for auth-gateway
├── jwt/
│   ├── validator.go      # JWT validation
│   └── extractor.go      # Token extraction
├── types/
│   ├── claims.go         # Claims structure
│   └── user.go           # User types
├── errors/
│   └── errors.go         # Standardized errors
├── go.mod
└── README.md
```

## 🔧 Configuration

### Environment Variables

```env
# JWT Secret (must be the same across all services)
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# Auth Gateway URL (for HTTP client)
AUTH_GATEWAY_URL=http://auth-gateway:8080
```

### Initialization

```go
import (
    "github.com/serphona/backend/go/libs/platform-auth/middleware"
    "os"
)

func main() {
    // Configure JWT secret
    jwtSecret := os.Getenv("JWT_SECRET")
    middleware.SetJWTSecret(jwtSecret)
    
    // Rest of the application...
}
```

## 📖 API Reference

### Middleware

#### `RequireAuth()`
Middleware that validates JWT and injects claims into context.

```go
router.Use(middleware.RequireAuth())
```

#### `RequireRole(role string)`
Middleware that requires a specific role.

```go
router.Use(middleware.RequireRole("admin"))
```

#### `GetClaimsFromContext(c *gin.Context)`
Extracts claims from request context.

```go
claims, err := middleware.GetClaimsFromContext(c)
```

### Client

#### `New(baseURL string)`
Creates a new HTTP client for auth-gateway.

```go
client := client.New("http://auth-gateway:8080")
```

#### `ValidateToken(token string)`
Validates token by calling auth-gateway.

```go
claims, err := client.ValidateToken(token)
```

#### `GetUserByID(userID string)`
Fetches user information.

```go
user, err := client.GetUserByID(userID)
```

### JWT

#### `ValidateToken(tokenString, secret string)`
Validates JWT locally (without calling auth-gateway).

```go
claims, err := jwt.ValidateToken(token, jwtSecret)
```

#### `ExtractTokenFromHeader(authHeader string)`
Extracts token from "Bearer xxx" header.

```go
token, err := jwt.ExtractTokenFromHeader(c.GetHeader("Authorization"))
```

## 🔒 Security

- ✅ JWT signature validation
- ✅ Expiration verification
- ✅ Refresh token support
- ✅ Custom claims (tenantID, role)
- ✅ Rate limiting on HTTP client

## 🧪 Tests

```bash
go test ./...
```

## 📝 Examples

See `examples/` folder for complete usage examples.

## 🤝 Contributing

This lib is maintained by the Serphona team. To contribute:

1. Create a branch
2. Make your changes
3. Add tests
4. Open a Pull Request

## 📚 Additional Documentation

- [Auth Gateway Service](../../services/auth-gateway/README.md)
- [Libs vs Services Guide](../../../docs/architecture/LIBS_VS_SERVICES.md)

---

**Version**: 1.0.0  
**License**: Proprietary
