# Platform Auth - Guia de Implementação

> 🔐 Guia completo para implementar a biblioteca platform-auth em seus microserviços

## 📋 O Que Foi Desenvolvido

A biblioteca `platform-auth` fornece os seguintes componentes:

### 1. **Tipos Compartilhados** (`types/`)
- ✅ `Claims` - Estrutura de claims JWT customizadas
- ✅ `User` - Representação de usuário
- ✅ `TokenResponse` - Resposta com tokens
- ✅ `AuthResponse` - Resposta completa de autenticação

### 2. **Erros Padronizados** (`errors/`)
- ✅ Erros comuns de autenticação
- ✅ Códigos de erro padronizados
- ✅ Tipo `AuthError` customizado

### 3. **Utilitários JWT** (`jwt/`)
- ✅ Validação de tokens JWT
- ✅ Extração de tokens de headers
- ✅ Configuração de secret

### 4. **Middleware** (`middleware/`)
- ✅ `RequireAuth()` - Valida JWT
- ✅ `RequireRole(role)` - Requer role específica
- ✅ `RequireAdmin()` - Requer admin/superadmin
- ✅ `RequireSuperAdmin()` - Requer superadmin
- ✅ Helpers para extrair dados do contexto

### 5. **Cliente HTTP** (`client/`)
- ✅ Cliente para comunicar com auth-gateway
- ✅ Validação de tokens
- ✅ Buscar informações de usuários
- ✅ Refresh de tokens
- ✅ Logout

---

## 🚀 Como Usar em Seus Services

### Passo 1: Adicionar Dependência

No `go.mod` do seu service:

```go
require (
    github.com/serphona/serphona/backend/go/libs/platform-auth v1.0.0
)
```

Execute:
```bash
go mod tidy
```

### Passo 2: Configurar Variáveis de Ambiente

No `.env` do seu service:

```env
# JWT Secret (DEVE SER O MESMO EM TODOS OS SERVICES)
JWT_SECRET=your-super-secret-jwt-key-min-32-chars

# Auth Gateway URL (opcional, para cliente HTTP)
AUTH_GATEWAY_URL=http://auth-gateway:8080
```

### Passo 3: Inicializar no Main

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
    // 1. Configurar JWT secret
    jwtSecret := os.Getenv("JWT_SECRET")
    if jwtSecret == "" {
        log.Fatal("JWT_SECRET não configurado")
    }
    authjwt.SetSecret(jwtSecret)
    
    // 2. Criar router
    router := gin.Default()
    
    // 3. Adicionar rotas...
    setupRoutes(router)
    
    // 4. Iniciar servidor
    router.Run(":8081")
}
```

### Passo 4: Proteger Rotas

```go
func setupRoutes(router *gin.Engine) {
    // Rotas públicas
    router.GET("/health", healthCheck)
    
    // Rotas protegidas
    api := router.Group("/api/v1")
    api.Use(middleware.RequireAuth())
    {
        // Qualquer usuário autenticado
        api.GET("/data", getData)
        api.POST("/items", createItem)
        
        // Somente admin
        admin := api.Group("/admin")
        admin.Use(middleware.RequireAdmin())
        {
            admin.GET("/users", listUsers)
            admin.DELETE("/users/:id", deleteUser)
        }
        
        // Somente superadmin
        superadmin := api.Group("/superadmin")
        superadmin.Use(middleware.RequireSuperAdmin())
        {
            superadmin.GET("/system", getSystemInfo)
        }
    }
}
```

### Passo 5: Usar Informações do Usuário

```go
func getData(c *gin.Context) {
    // Opção 1: Extrair claims completas
    claims, err := middleware.GetClaimsFromContext(c)
    if err != nil {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }
    
    log.Printf("User: %s (%s)", claims.Name, claims.Email)
    log.Printf("Tenant: %s", claims.TenantID)
    log.Printf("Role: %s", claims.Role)
    
    // Opção 2: Extrair dados específicos
    userID, _ := middleware.GetUserIDFromContext(c)
    tenantID, _ := middleware.GetTenantIDFromContext(c)
    
    // Opção 3: Usar valores diretos do contexto
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

## 📚 Exemplos de Uso

### Exemplo 1: Billing Service

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
        // Listar faturas do tenant do usuário
        api.GET("/invoices", func(c *gin.Context) {
            tenantID, _ := middleware.GetTenantIDFromContext(c)
            invoices := getInvoicesByTenant(tenantID)
            c.JSON(200, invoices)
        })
        
        // Criar assinatura
        api.POST("/subscriptions", func(c *gin.Context) {
            claims, _ := middleware.GetClaimsFromContext(c)
            
            // Validar que o usuário pode criar assinatura
            if !claims.IsAdmin() {
                c.JSON(403, gin.H{"error": "Admin required"})
                return
            }
            
            // Criar assinatura...
        })
    }
    
    router.Run(":8081")
}
```

### Exemplo 2: Tenant Manager Service

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
        // Obter tenant atual
        api.GET("/current", func(c *gin.Context) {
            tenantID, _ := middleware.GetTenantIDFromContext(c)
            tenant := getTenantByID(tenantID)
            c.JSON(200, tenant)
        })
        
        // Listar membros do tenant (somente admin)
        api.GET("/members", middleware.RequireAdmin(), func(c *gin.Context) {
            tenantID, _ := middleware.GetTenantIDFromContext(c)
            members := getMembersByTenant(tenantID)
            c.JSON(200, members)
        })
        
        // Gerenciar todos os tenants (somente superadmin)
        api.GET("/all", middleware.RequireSuperAdmin(), func(c *gin.Context) {
            tenants := getAllTenants()
            c.JSON(200, tenants)
        })
    }
    
    router.Run(":8082")
}
```

### Exemplo 3: Usando Cliente HTTP

```go
package main

import (
    "github.com/serphona/serphona/backend/go/libs/platform-auth/client"
)

func main() {
    // Criar cliente
    authClient := client.New("http://auth-gateway:8080")
    
    // Validar token chamando auth-gateway
    token := "eyJhbGc..."
    claims, err := authClient.ValidateToken(token)
    if err != nil {
        log.Fatal("Token inválido:", err)
    }
    
    log.Printf("User: %s", claims.Email)
    
    // Buscar informações do usuário
    user, err := authClient.GetMe(token)
    if err != nil {
        log.Fatal("Erro ao buscar usuário:", err)
    }
    
    log.Printf("User: %+v", user)
    
    // Refresh token
    newTokens, err := authClient.RefreshToken(refreshToken)
    if err != nil {
        log.Fatal("Erro ao renovar token:", err)
    }
    
    log.Printf("New access token: %s", newTokens.AccessToken)
}
```

---

## 🔒 Segurança

### Validação Local vs Gateway

**Validação Local (Recomendado):**
```go
// Mais rápido, não faz chamada HTTP
claims, err := authjwt.ValidateToken(token)
```

**Validação via Gateway:**
```go
// Mais seguro, verifica se sessão ainda é válida
authClient := client.New("http://auth-gateway:8080")
claims, err := authClient.ValidateToken(token)
```

### Recomendações:

1. ✅ Use **validação local** para a maioria das requests
2. ✅ Use **validação via gateway** para operações sensíveis
3. ✅ Sempre use HTTPS em produção
4. ✅ Nunca exponha o JWT_SECRET
5. ✅ Implemente rate limiting
6. ✅ Valide input do usuário

---

## 🧪 Testes

### Testar Middleware

```go
func TestRequireAuth(t *testing.T) {
    authjwt.SetSecret("test-secret-key-minimum-32-chars")
    
    router := gin.Default()
    router.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "success"})
    })
    
    // Criar token válido
    token := createTestToken()
    
    // Request com token
    req := httptest.NewRequest("GET", "/protected", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    assert.Equal(t, 200, w.Code)
}
```

---

## 🐛 Troubleshooting

### Erro: "JWT secret not configured"

**Solução:** Configure o secret antes de usar:
```go
authjwt.SetSecret(os.Getenv("JWT_SECRET"))
```

### Erro: "Missing authentication token"

**Solução:** Certifique-se de enviar o header:
```bash
curl -H "Authorization: Bearer <token>" http://localhost:8081/api/v1/data
```

### Erro: "Token expired"

**Solução:** Use refresh token para renovar:
```go
newTokens, err := authClient.RefreshToken(refreshToken)
```

### Erro: "Insufficient permissions"

**Solução:** Verifique a role do usuário:
- `user` - usuário comum
- `admin` - administrador do tenant
- `superadmin` - super administrador da plataforma

---

## 📝 Checklist de Implementação

Ao adicionar platform-auth em um novo service:

- [ ] Adicionar dependência no go.mod
- [ ] Executar `go mod tidy`
- [ ] Adicionar JWT_SECRET no .env
- [ ] Configurar secret no main.go
- [ ] Adicionar middleware RequireAuth() nas rotas protegidas
- [ ] Extrair userID/tenantID do contexto onde necessário
- [ ] Implementar tratamento de erros apropriado
- [ ] Testar com token válido e inválido
- [ ] Documentar endpoints protegidos no README
- [ ] Configurar CORS se necessário

---

## 🔗 Links Úteis

- [Auth Gateway README](../../services/auth-gateway/README.md)
- [Libs vs Services Guide](../../../docs/architecture/LIBS_VS_SERVICES.md)
- [Platform Auth README](./README.md)
- [Exemplo Completo](./examples/basic_usage.go)

---

## 📞 Suporte

Para dúvidas ou problemas:
1. Verifique este guia
2. Veja os exemplos em `examples/`
3. Consulte a documentação do auth-gateway
4. Abra uma issue no repositório

---

**Última atualização**: 29/11/2025  
**Versão da Lib**: 1.0.0
