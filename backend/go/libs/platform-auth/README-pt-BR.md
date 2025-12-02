# Platform Auth Library

> 🔐 Biblioteca compartilhada de autenticação para os microserviços do Serphona.

## 📋 Propósito

Esta biblioteca fornece componentes reutilizáveis de autenticação para serem usados por todos os services do Serphona, **exceto** o `auth-gateway` que é quem implementa a lógica de autenticação completa.

## 🎯 Responsabilidades

A `platform-auth` **NÃO** implementa:
- ❌ Login/Logout
- ❌ Registro de usuários
- ❌ Gestão de banco de dados de usuários
- ❌ OAuth providers
- ❌ Emissão de tokens

A `platform-auth` **FORNECE**:
- ✅ Middleware de validação JWT
- ✅ Cliente HTTP para chamar auth-gateway
- ✅ Tipos compartilhados (Claims, User, etc)
- ✅ Utilitários de JWT
- ✅ Erros padronizados

## 📦 Instalação

```bash
go get github.com/serphona/backend/go/libs/platform-auth
```

## 🚀 Uso

### 1. Middleware de Autenticação

Use em qualquer service para proteger rotas:

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/serphona/backend/go/libs/platform-auth/middleware"
)

func main() {
    router := gin.Default()
    
    // Rotas públicas
    router.GET("/health", healthHandler)
    
    // Rotas protegidas
    protected := router.Group("/api/v1")
    protected.Use(middleware.RequireAuth())
    {
        protected.GET("/billing/invoices", getInvoices)
        protected.GET("/tenants/current", getCurrentTenant)
    }
    
    router.Run(":8081")
}
```

### 2. Extrair Informações do Usuário

```go
func getInvoices(c *gin.Context) {
    // Extrai claims do contexto (injetado pelo middleware)
    userID := c.GetString("userID")
    tenantID := c.GetString("tenantID")
    role := c.GetString("role")
    
    // Ou use o helper
    claims, err := middleware.GetClaimsFromContext(c)
    if err != nil {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }
    
    // Use as informações
    invoices := getInvoicesForTenant(claims.TenantID)
    c.JSON(200, invoices)
}
```

### 3. Cliente HTTP para Auth Gateway

```go
package main

import (
    "github.com/serphona/backend/go/libs/platform-auth/client"
)

func main() {
    // Criar cliente
    authClient := client.New("http://auth-gateway:8080")
    
    // Validar token
    claims, err := authClient.ValidateToken(token)
    if err != nil {
        // Token inválido
    }
    
    // Obter informações do usuário
    user, err := authClient.GetUserByID(userID)
}
```

### 4. Validação Manual de JWT

```go
import "github.com/serphona/backend/go/libs/platform-auth/jwt"

// Validar token manualmente
claims, err := jwt.ValidateToken(tokenString, jwtSecret)
if err != nil {
    // Token inválido
}

// Extrair token do header Authorization
token, err := jwt.ExtractTokenFromHeader(authHeader)
```

## 📁 Estrutura

```
platform-auth/
├── middleware/
│   ├── auth.go           # Middleware RequireAuth()
│   └── context.go        # Helpers para context
├── client/
│   └── auth_client.go    # Cliente HTTP para auth-gateway
├── jwt/
│   ├── validator.go      # Validação de JWT
│   └── extractor.go      # Extração de token
├── types/
│   ├── claims.go         # Estrutura de claims
│   └── user.go           # Tipos de usuário
├── errors/
│   └── errors.go         # Erros padronizados
├── go.mod
└── README.md
```

## 🔧 Configuração

### Variáveis de Ambiente

```env
# JWT Secret (deve ser o mesmo em todos os services)
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# Auth Gateway URL (para cliente HTTP)
AUTH_GATEWAY_URL=http://auth-gateway:8080
```

### Inicialização

```go
import (
    "github.com/serphona/backend/go/libs/platform-auth/middleware"
    "os"
)

func main() {
    // Configurar JWT secret
    jwtSecret := os.Getenv("JWT_SECRET")
    middleware.SetJWTSecret(jwtSecret)
    
    // Resto da aplicação...
}
```

## 📖 API Reference

### Middleware

#### `RequireAuth()`
Middleware que valida JWT e injeta claims no contexto.

```go
router.Use(middleware.RequireAuth())
```

#### `RequireRole(role string)`
Middleware que requer uma role específica.

```go
router.Use(middleware.RequireRole("admin"))
```

#### `GetClaimsFromContext(c *gin.Context)`
Extrai claims do contexto da request.

```go
claims, err := middleware.GetClaimsFromContext(c)
```

### Client

#### `New(baseURL string)`
Cria novo cliente HTTP para auth-gateway.

```go
client := client.New("http://auth-gateway:8080")
```

#### `ValidateToken(token string)`
Valida token chamando auth-gateway.

```go
claims, err := client.ValidateToken(token)
```

#### `GetUserByID(userID string)`
Busca informações do usuário.

```go
user, err := client.GetUserByID(userID)
```

### JWT

#### `ValidateToken(tokenString, secret string)`
Valida JWT localmente (sem chamar auth-gateway).

```go
claims, err := jwt.ValidateToken(token, jwtSecret)
```

#### `ExtractTokenFromHeader(authHeader string)`
Extrai token do header "Bearer xxx".

```go
token, err := jwt.ExtractTokenFromHeader(c.GetHeader("Authorization"))
```

## 🔒 Segurança

- ✅ Validação de assinatura JWT
- ✅ Verificação de expiração
- ✅ Suporte a refresh tokens
- ✅ Claims customizados (tenantID, role)
- ✅ Rate limiting no cliente HTTP

## 🧪 Testes

```bash
go test ./...
```

## 📝 Exemplos

Ver pasta `examples/` para exemplos completos de uso.

## 🤝 Contribuindo

Esta lib é mantida pela equipe Serphona. Para contribuir:

1. Crie uma branch
2. Faça suas alterações
3. Adicione testes
4. Abra um Pull Request

## 📚 Documentação Adicional

- [Auth Gateway Service](../../services/auth-gateway/README.md)
- [Libs vs Services Guide](../../../docs/architecture/LIBS_VS_SERVICES.md)

---

**Versão**: 1.0.0  
**Licença**: Proprietary
