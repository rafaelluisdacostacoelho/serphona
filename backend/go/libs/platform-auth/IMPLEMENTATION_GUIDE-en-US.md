# Platform Auth - Implementation Guide

Complete guide to implement the platform-auth library in your microservices.

## What Has Been Developed

### Shared Types (`types/`)
- `Claims` — custom JWT claims (user, tenant, role, session)
- `User` — user representation
- `TokenResponse` — token payload (access/refresh)
- `AuthResponse` — combined user + tokens

### Standardized Errors (`errors/`)
- Common authentication errors
- Error codes
- Custom `AuthError` type

### JWT Utilities (`jwt/`)
- JWT token validation
- Extraction from `Authorization` header
- Secret configuration

### Middleware (`middleware/`)
- `RequireAuth()` — validates JWT and injects claims
- `RequireRole(role)` — enforces a specific role
- `RequireAdmin()` / `RequireSuperAdmin()` — convenience guards
- Helpers to read data from Gin context

### HTTP Client (`client/`)
- Client to communicate with auth-gateway
- Token validation
- Fetch user information
- Token refresh
- Logout

---

## How to Use in Your Services

### Step 1: Add Dependency

`go.mod`
```go
require github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth v1.0.0
```

Run:
```bash
go mod tidy
```

### Step 2: Configure Environment Variables

```env
JWT_SECRET=your-super-secret-jwt-key-min-32-chars
AUTH_GATEWAY_URL=http://auth-gateway:8080
TENANT_ID_HEADER=X-Tenant-Id
TRACE_REQUEST_HEADER=X-Request-Id
TLS_CA=/etc/ssl/certs/ca.pem
TLS_CERT=/etc/ssl/certs/client.pem
TLS_KEY=/etc/ssl/private/client.key
```

### Step 3: Initialize in `main`

```go
func main() {
    authjwt.MustSetSecretFromEnv() // panics if JWT_SECRET is missing

    router := gin.Default()
    setupRoutes(router)
    router.Run(":8081")
}
```

### Step 4: Protect Routes

```go
func setupRoutes(router *gin.Engine) {
    router.GET("/health", healthCheck)

    api := router.Group("/api/v1")
    api.Use(middleware.RequireAuth())
    {
        api.GET("/data", getData)
        api.POST("/items", createItem)

        admin := api.Group("/admin")
        admin.Use(middleware.RequireAdmin())
        admin.GET("/users", listUsers)
        admin.DELETE("/users/:id", deleteUser)

        superadmin := api.Group("/superadmin")
        superadmin.Use(middleware.RequireSuperAdmin())
        superadmin.GET("/system", getSystemInfo)
    }
}
```

### Step 5: Use User Information

```go
func getData(c *gin.Context) {
    claims, err := middleware.GetClaimsFromContext(c)
    if err != nil {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }

    userID, _ := middleware.GetUserIDFromContext(c)
    tenantID, _ := middleware.GetTenantIDFromContext(c)

    c.JSON(200, gin.H{
        "userId":   userID,
        "tenantId": tenantID,
        "email":    claims.Email,
        "role":     claims.Role,
    })
}
```

### Step 6: Enforce Tenant/RLS and propagate to DB/Kafka/HTTP

```go
func handleWrite(ctx context.Context, payloadTenant string) error {
    tenantID, err := middleware.TenantIDFromContext(ctx)
    if err != nil {
        return err
    }
    if err := middleware.EnforceTenant(ctx, payloadTenant); err != nil {
        return err // fail on cross-tenant attempts
    }

    // Example: add tenant to Kafka headers
    headers := []kafka.Header{{Key: middleware.TenantIDHeader, Value: []byte(tenantID)}}
    // producer.WriteMessages(ctx, kafka.Message{Headers: headers, Value: ...})

    // Example: outbound HTTP request carrying tenant
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, downstreamURL, body)
    req.Header = middleware.EnsureTenantHeader(req.Header, tenantID)
    _, err = http.DefaultClient.Do(req)
    return err
}
```

---

## Usage Examples

### Example 1: Billing Service

```go
func main() {
    authjwt.SetSecret(os.Getenv("JWT_SECRET"))

    router := gin.Default()
    api := router.Group("/api/v1/billing")
    api.Use(middleware.RequireAuth())

    api.GET("/invoices", func(c *gin.Context) {
        tenantID, _ := middleware.GetTenantIDFromContext(c)
        invoices := getInvoicesByTenant(tenantID)
        c.JSON(200, invoices)
    })

    api.POST("/subscriptions", func(c *gin.Context) {
        claims, _ := middleware.GetClaimsFromContext(c)
        if !claims.IsAdmin() {
            c.JSON(403, gin.H{"error": "Admin required"})
            return
        }
        // create subscription...
    })

    router.Run(":8081")
}
```

### Example 2: Tenant Manager Service

```go
func main() {
    router := gin.Default()

    api := router.Group("/api/v1/tenants")
    api.Use(middleware.RequireAuth())
    api.GET("/current", func(c *gin.Context) {
        tenantID, _ := middleware.GetTenantIDFromContext(c)
        tenant := getTenantByID(tenantID)
        c.JSON(200, tenant)
    })

    api.GET("/members", middleware.RequireAdmin(), func(c *gin.Context) {
        tenantID, _ := middleware.GetTenantIDFromContext(c)
        members := getMembersByTenant(tenantID)
        c.JSON(200, members)
    })

    api.GET("/all", middleware.RequireSuperAdmin(), func(c *gin.Context) {
        tenants := getAllTenants()
        c.JSON(200, tenants)
    })

    router.Run(":8082")
}
```

### Example 3: Using HTTP Client

```go
authClient := client.New("http://auth-gateway:8080")

claims, err := authClient.ValidateToken(token)
if err != nil {
    log.Fatal("Invalid token:", err)
}

user, err := authClient.GetMe(token)
if err != nil {
    log.Fatal("Error fetching user:", err)
}

newTokens, err := authClient.RefreshToken(refreshToken)
if err != nil {
    log.Fatal("Error refreshing token:", err)
}

// or load URL from environment
authClient, err := client.NewFromEnv()
if err != nil {
    log.Fatal(err)
}
```

### Response envelope examples (Gin / chi / gRPC)

Gin:

```go
api := router.Group("/api")
api.Use(middleware.RequireAuth())
api.GET("/invoices", func(c *gin.Context) {
    invoices := []gin.H{{"id": "inv-1"}}
    response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, invoices, response.WithPagination(response.Pagination{Page: 1, PageSize: 10, Total: 1, TotalPages: 1}))
})
api.POST("/invoices", func(c *gin.Context) {
    response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_INPUT", "missing fields", nil)
})
```

net/http (chi-compatible):

```go
mux := http.NewServeMux()
mux.Handle("/reports", middleware.RequireAuthHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    payload := []map[string]any{{"id": "r1"}}
    response.WriteSuccess(r.Context(), w, http.StatusOK, payload)
})))
```

gRPC: add the interceptor so request/trace IDs flow via metadata and auth errors map to gRPC status codes:

```go
grpcServer := grpc.NewServer(grpc.UnaryInterceptor(middleware.UnaryAuthInterceptor("scope:read")))
```

See runnable samples in `examples/envelope_gin` and `examples/envelope_http`.
---

## Security

### Local Validation vs Gateway

- **Local validation (recommended):** `authjwt.ValidateToken(token)` — faster, no HTTP call.
- **Gateway validation:** `client.ValidateToken(token)` — checks session validity on auth-gateway.

### Recommendations

1. Use local validation for most requests.
2. Use gateway validation for sensitive operations.
3. Always use HTTPS in production.
4. Never expose or log `JWT_SECRET`.
5. Apply rate limiting where appropriate.
6. Validate incoming data before use.

---

## Tests

Run all tests:
```bash
go test ./...
```

Example (middleware):
```go
router := gin.Default()
router.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
    c.JSON(200, gin.H{"message": "ok"})
})
```

---

## Troubleshooting

- **"JWT secret not configured"** — call `authjwt.SetSecret` before using middleware/JWT helpers.
- **"Missing authentication token"** — send `Authorization: Bearer <token>` header.
- **"Token expired"** — refresh the token using `client.RefreshToken`.
- **"Insufficient permissions"** — verify the user role.

---

## Document protected endpoints

- Keep a short table of the routes that use `RequireAuth`, `RequireAdmin`, `RequireSuperAdmin`, or `RequireRole` so reviewers and auditors know what is locked down.
- Include the required role/tenant scope and the service that owns the route. Example:

| Route | Method | Middleware | Role | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/billing/invoices` | GET | `RequireAuth` | any | user/tenant scoped |
| `/api/v1/billing/users/:id` | DELETE | `RequireAdmin` | admin | admin only |
| `/api/v1/system/tenants` | GET | `RequireSuperAdmin` | superadmin | platform control |

Keep this list close to your service README or runbook and update it when new routes are added.

## Configure CORS

Enable CORS on the API server to allow browser clients (console/MFEs) to call protected endpoints with the `Authorization` header.

```go
import "github.com/gin-contrib/cors"
import "time"

corsCfg := cors.Config{
    AllowOrigins:     []string{"http://localhost:5173", "https://console.serphona.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Authorization", "Content-Type", "Accept", "X-Request-ID"},
    ExposeHeaders:    []string{"X-Request-ID"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}

router := gin.Default()
router.Use(cors.New(corsCfg))
```

Adjust `AllowOrigins` per environment (dev vs prod) and keep `Authorization` in `AllowHeaders` so JWT bearer headers flow correctly.

---

## Useful Links

- Auth Gateway README: `../../services/auth-gateway/README.md`
- Libs vs Services Guide: `../../../docs/architecture/LIBS_VS_SERVICES.md`
- Platform Auth README: `./README-en-US.md`
- Complete example: `./examples/basic_usage.go`

---

## Support

1. Check this guide and the examples.
2. Consult the auth-gateway documentation.
3. Open an issue in the repository if needed.

---

Last updated: December 2025  
Library Version: 1.0.0
