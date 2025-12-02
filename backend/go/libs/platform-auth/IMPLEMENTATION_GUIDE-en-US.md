# Platform Auth - Implementation Guide

> 🔐 Complete guide to implement the platform-auth library in your microservices

## 📋 What Has Been Developed

The `platform-auth` library provides the following components:

### 1. **Shared Types** (`types/`)
- ✅ `Claims` - Custom JWT claims structure
- ✅ `User` - User representation
- ✅ `TokenResponse` - Token response
- ✅ `AuthResponse` - Complete authentication response

### 2. **Standardized Errors** (`errors/`)
- ✅ Common authentication errors
- ✅ Standardized error codes
- ✅ Custom `AuthError` type

### 3. **JWT Utilities** (`jwt/`)
- ✅ JWT token validation
- ✅ Token extraction from headers
- ✅ Secret configuration

### 4. **Middleware** (`middleware/`)
- ✅ `RequireAuth()` - Validates JWT
- ✅ `RequireRole(role)` - Requires specific role
- ✅ `RequireAdmin()` - Requires admin/superadmin
- ✅ `RequireSuperAdmin()` - Requires superadmin
- ✅ Helpers to extract data from context

### 5. **HTTP Client** (`client/`)
- ✅ Client to communicate with auth-gateway
- ✅ Token validation
- ✅ Fetch user information
- ✅ Token refresh
- ✅ Logout

---

## 🚀 How to Use in Your Services

### Step 1: Add Dependency

In your service's `go.mod`:

```go
require (
    github.com/serphona/serphona/backend/go/libs/platform-auth v1.0.0
)
```

Execute:
```bash
go mod tidy
```

### Step 2: Configure Environment Variables

In your service's `.env`:

```env
# JWT Secret (MUST BE THE SAME IN ALL SERVICES)
JWT_SECRET=your-super-secret-jwt-key-min-32-chars

# Auth Gateway URL (optional, for HTTP client)
AUTH_GATEWAY_URL=http://auth-gateway:8080
```

### Step 3: Initialize in Main

```go
package main

import (
    "log"
    "os"
    
    "github.com/gin-gonic/gin"
    authjwt "github.com/serphona/serphona/backend/go/libs/platform-auth/jwt"
    "github.com/serphona/serphona/backend/go/libs/platform-auth/middleware"
)

func main() {
    // 1. Configure JWT secret
    jwtSecret := os.Getenv("JWT_SECRET")
    if jwtSecret == "" {
        log.Fatal("JWT_SECRET not configured")
    }
    authjwt.SetSecret(jwtSecret)
    
    // 2. Create router
    router := gin.Default()
    
    // 3. Add routes...
    setupRoutes(router)
    
    // 4. Start server
    router.Run(":8081")
}
```

### Step 4: Protect Routes

```go
func setupRoutes(router *gin.Engine) {
    // Public routes
    router.GET("/health", healthCheck)
    
    // Protected routes
    api := router.Group("/api/v1")
    api.Use(middleware.RequireAuth())
    {
        // Any authenticated user
        api.GET("/data", getData)
        api.POST("/items", createItem)
        
        // Admin only
        admin := api.Group("/admin")
        admin.Use(middleware.RequireAdmin())
        {
            admin.GET("/users", listUsers)
            admin.DELETE("/users/:id", deleteUser)
        }
        
        // Superadmin only
        superadmin := api.Group("/superadmin")
        superadmin.Use(middleware.RequireSuperAdmin())
        {
            superadmin.GET("/system", getSystemInfo)
        }
    }
}
```

### Step 5: Use User Information

```go
func getData(c *gin.Context) {
    // Option 1: Extract complete claims
    claims, err := middleware.GetClaimsFromContext(c)
    if err != nil {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }
    
    log.Printf("User: %s (%s)", claims.Name, claims.Email)
    log.Printf("Tenant: %s", claims.TenantID)
    log.Printf("Role: %s", claims.Role)
    
    // Option 2: Extract specific data
    userID, _ := middleware.GetUserIDFromContext(c)
    tenantID, _ := middleware.GetTenantIDFromContext(c)
    
    // Option 3: Use values directly from context
    email := c.GetString("email")
    role := c.GetString("role")
    
    c.JSON(200, gin.H{
        "userId": userID,
        "tenantId": tenantID,
        "email": email,
        "role": role,
    })
}
```

---

## 📚 Usage Examples

### Example 1: Billing Service

```go
package main

import (
    "github.com/gin-gonic/gin"
    authjwt "github.com/serphona/serphona/backend/go/libs/platform-auth/jwt"
    "github.com/serphona/serphona/backend/go/libs/platform-auth/middleware"
)

func main() {
    authjwt.SetSecret(os.Getenv("JWT_SECRET"))
    
    router := gin.Default()
    
    api := router.Group("/api/v1/billing")
    api.Use(middleware.RequireAuth())
    {
        // List invoices for user's tenant
        api.GET("/invoices", func(c *gin.Context) {
            tenantID, _ := middleware.GetTenantIDFromContext(c)
            invoices := getInvoicesByTenant(tenantID)
            c.JSON(200, invoices)
        })
        
        // Create subscription
        api.POST("/subscriptions", func(c *gin.Context) {
            claims, _ := middleware.GetClaimsFromContext(c)
            
            // Validate that user can create subscription
            if !claims.IsAdmin() {
                c.JSON(403, gin.H{"error": "Admin required"})
                return
            }
            
            // Create subscription...
        })
    }
    
    router.Run(":8081")
}
```

### Example 2: Tenant Manager Service

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/serphona/serphona/backend/go/libs/platform-auth/middleware"
)

func main() {
    router := gin.Default()
    
    api := router.Group("/api/v1/tenants")
    api.Use(middleware.RequireAuth())
    {
        // Get current tenant
        api.GET("/current", func(c *gin.Context) {
            tenantID, _ := middleware.GetTenantIDFromContext(c)
            tenant := getTenantByID(tenantID)
            c.JSON(200, tenant)
        })
        
        // List tenant members (admin only)
        api.GET("/members", middleware.RequireAdmin(), func(c *gin.Context) {
            tenantID, _ := middleware.GetTenantIDFromContext(c)
            members := getMembersByTenant(tenantID)
            c.JSON(200, members)
        })
        
        // Manage all tenants (superadmin only)
        api.GET("/all", middleware.RequireSuperAdmin(), func(c *gin.Context) {
            tenants := getAllTenants()
            c.JSON(200, tenants)
        })
    }
    
    router.Run(":8082")
}
```

### Example 3: Using HTTP Client

```go
package main

import (
    "github.com/serphona/serphona/backend/go/libs/platform-auth/client"
)

func main() {
    // Create client
    authClient := client.New("http://auth-gateway:8080")
    
    // Validate token by calling auth-gateway
    token := "eyJhbGc..."
    claims, err := authClient.ValidateToken(token)
    if err != nil {
        log.Fatal("Invalid token:", err)
    }
    
    log.Printf("User: %s", claims.Email)
    
    // Fetch user information
    user, err := authClient.GetMe(token)
    if err != nil {
        log.Fatal("Error fetching user:", err)
    }
    
    log.Printf("User: %+v", user)
    
    // Refresh token
    newTokens, err := authClient.RefreshToken(refreshToken)
    if err != nil {
        log.Fatal("Error refreshing token:", err)
    }
    
    log.Printf("New access token: %s", newTokens.AccessToken)
}
```

---

## 🔒 Security

### Local Validation vs Gateway

**Local Validation (Recommended):**
```go
// Faster, doesn't make HTTP call
claims, err := authjwt.ValidateToken(token)
```

**Gateway Validation:**
```go
// More secure, checks if session is still valid
authClient := client.New("http://auth-gateway:8080")
claims, err := authClient.ValidateToken(token)
```

### Recommendations:

1. ✅ Use **local validation** for most requests
2. ✅ Use **gateway validation** for sensitive operations
3. ✅ Always use HTTPS in production
4. ✅ Never expose JWT_SECRET
5. ✅ Implement rate limiting
6. ✅ Validate user input

---

## 🧪 Tests

### Testing Middleware

```go
func TestRequireAuth(t *testing.T) {
    authjwt.SetSecret("test-secret-key-minimum-32-chars")
    
    router := gin.Default()
    router.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "success"})
    })
    
    // Create valid token
    token := createTestToken()
    
    // Request with token
    req := httptest.NewRequest("GET", "/protected", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    assert.Equal(t, 200, w.Code)
}
```

---

## 🐛 Troubleshooting

### Error: "JWT secret not configured"

**Solution:** Configure the secret before use:
```go
authjwt.SetSecret(os.Getenv("JWT_SECRET"))
```

### Error: "Missing authentication token"

**Solution:** Make sure to send the header:
```bash
curl -H "Authorization: Bearer <token>" http://localhost:8081/api/v1/data
```

### Error: "Token expired"

**Solution:** Use refresh token to renew:
```go
newTokens, err := authClient.RefreshToken(refreshToken)
```

### Error: "Insufficient permissions"

**Solution:** Check user role:
- `user` - regular user
- `admin` - tenant administrator
- `superadmin` - platform super administrator

---

## 📝 Implementation Checklist

When adding platform-auth to a new service:

- [ ] Add dependency to go.mod
- [ ] Run `go mod tidy`
- [ ] Add JWT_SECRET to .env
- [ ] Configure secret in main.go
- [ ] Add RequireAuth() middleware to protected routes
- [ ] Extract userID/tenantID from context where needed
- [ ] Implement appropriate error handling
- [ ] Test with valid and invalid tokens
- [ ] Document protected endpoints in README
- [ ] Configure CORS if necessary

---

## 🔗 Useful Links

- [Auth Gateway README](../../services/auth-gateway/README.md)
- [Libs vs Services Guide](../../../docs/architecture/LIBS_VS_SERVICES.md)
- [Platform Auth README](./README.md)
- [Complete Example](./examples/basic_usage.go)

---

## 📞 Support

For questions or issues:
1. Check this guide
2. See examples in `examples/`
3. Consult auth-gateway documentation
4. Open an issue in the repository

---

**Last updated**: November 29, 2025  
**Library Version**: 1.0.0
