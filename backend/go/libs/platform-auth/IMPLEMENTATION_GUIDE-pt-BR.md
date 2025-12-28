# Platform Auth - Guia de Implementacao

Guia completo para usar a biblioteca platform-auth nos seus microsservicos.

## O que ja foi desenvolvido

### Tipos compartilhados (`types/`)
- `Claims` — claims customizadas do JWT (usuario, tenant, role, sessao)
- `User` — representacao de usuario
- `TokenResponse` — payload de tokens (access/refresh)
- `AuthResponse` — usuario + tokens

### Erros padronizados (`errors/`)
- Erros comuns de autenticacao
- Codigos de erro
- Tipo `AuthError`

### Utilitarios JWT (`jwt/`)
- Validacao de token JWT
- Extracao do header `Authorization`
- Configuracao do secret

### Middleware (`middleware/`)
- `RequireAuth()` — valida JWT e injeta claims
- `RequireRole(role)` — exige role especifica
- `RequireAdmin()` / `RequireSuperAdmin()` — atalhos
- Helpers para ler dados do contexto do Gin

### Cliente HTTP (`client/`)
- Cliente para comunicar com o auth-gateway
- Validacao de token
- Buscar informacoes de usuario
- Refresh de token
- Logout

---

## Como usar nos servicos

### Passo 1: Adicionar dependencia

`go.mod`
```go
require github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth v1.0.0
```

Execute:
```bash
go mod tidy
```

### Passo 2: Configurar variaveis de ambiente

```env
JWT_SECRET=sua-chave-jwt-min-32-caracteres
AUTH_GATEWAY_URL=http://auth-gateway:8080
TENANT_ID_HEADER=X-Tenant-Id
TRACE_REQUEST_HEADER=X-Request-Id
TLS_CA=/etc/ssl/certs/ca.pem
TLS_CERT=/etc/ssl/certs/client.pem
TLS_KEY=/etc/ssl/private/client.key
```

### Passo 3: Inicializar no `main`

```go
func main() {
    authjwt.MustSetSecretFromEnv() // gera panic se JWT_SECRET estiver ausente

    router := gin.Default()
    configurarRotas(router)
    router.Run(":8081")
}
```

### Passo 4: Proteger rotas

```go
func configurarRotas(router *gin.Engine) {
    router.GET("/health", healthCheck)

    api := router.Group("/api/v1")
    api.Use(middleware.RequireAuth())
    {
        api.GET("/dados", getDados)
        api.POST("/itens", criarItem)

        admin := api.Group("/admin")
        admin.Use(middleware.RequireAdmin())
        admin.GET("/usuarios", listarUsuarios)
        admin.DELETE("/usuarios/:id", removerUsuario)

        superadmin := api.Group("/superadmin")
        superadmin.Use(middleware.RequireSuperAdmin())
        superadmin.GET("/sistema", getInfoSistema)
    }
}
```

### Passo 5: Usar informacoes do usuario

```go
func getDados(c *gin.Context) {
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

### Passo 6: Aplicar Tenant/RLS e propagar para DB/Kafka/HTTP

```go
func escreverDados(ctx context.Context, tenantDoPayload string) error {
    tenantID, err := middleware.TenantIDFromContext(ctx)
    if err != nil {
        return err
    }
    if err := middleware.EnforceTenant(ctx, tenantDoPayload); err != nil {
        return err // impede cross-tenant
    }

    // Exemplo: adicionar tenant em headers do Kafka
    headers := []kafka.Header{{Key: middleware.TenantIDHeader, Value: []byte(tenantID)}}
    // producer.WriteMessages(ctx, kafka.Message{Headers: headers, Value: ...})

    // Exemplo: requisicao HTTP de saida com tenant
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, downstreamURL, body)
    req.Header = middleware.EnsureTenantHeader(req.Header, tenantID)
    _, err = http.DefaultClient.Do(req)
    return err
}
```

---

## Exemplos de uso

### Exemplo 1: Billing Service

```go
func main() {
    authjwt.SetSecret(os.Getenv("JWT_SECRET"))

    router := gin.Default()
    api := router.Group("/api/v1/billing")
    api.Use(middleware.RequireAuth())

    api.GET("/faturas", func(c *gin.Context) {
        tenantID, _ := middleware.GetTenantIDFromContext(c)
        invoices := buscarFaturasPorTenant(tenantID)
        c.JSON(200, invoices)
    })

    api.POST("/assinaturas", func(c *gin.Context) {
        claims, _ := middleware.GetClaimsFromContext(c)
        if !claims.IsAdmin() {
            c.JSON(403, gin.H{"error": "Admin required"})
            return
        }
        // criar assinatura...
    })

    router.Run(":8081")
}
```

### Exemplo 2: Tenant Manager Service

```go
func main() {
    router := gin.Default()

    api := router.Group("/api/v1/tenants")
    api.Use(middleware.RequireAuth())
    api.GET("/atual", func(c *gin.Context) {
        tenantID, _ := middleware.GetTenantIDFromContext(c)
        tenant := buscarTenantPorID(tenantID)
        c.JSON(200, tenant)
    })

    api.GET("/membros", middleware.RequireAdmin(), func(c *gin.Context) {
        tenantID, _ := middleware.GetTenantIDFromContext(c)
        membros := listarMembrosPorTenant(tenantID)
        c.JSON(200, membros)
    })

    api.GET("/todos", middleware.RequireSuperAdmin(), func(c *gin.Context) {
        tenants := listarTodosTenants()
        c.JSON(200, tenants)
    })

    router.Run(":8082")
}
```

### Exemplo 3: Usando o cliente HTTP

```go
authClient := client.New("http://auth-gateway:8080")

claims, err := authClient.ValidateToken(token)
if err != nil {
    log.Fatal("Token invalido:", err)
}

usuario, err := authClient.GetMe(token)
if err != nil {
    log.Fatal("Erro ao buscar usuario:", err)
}

novosTokens, err := authClient.RefreshToken(refreshToken)
if err != nil {
    log.Fatal("Erro ao renovar token:", err)
}

// ou carregando a URL do ambiente
authClient, err := client.NewFromEnv()
if err != nil {
    log.Fatal(err)
}
```

---

## Seguranca

### Validacao local vs gateway

- **Validacao local (recomendada):** `authjwt.ValidateToken(token)` — mais rapida, sem chamada HTTP.
- **Validacao via gateway:** `client.ValidateToken(token)` — verifica validade da sessao no auth-gateway.

### Recomendações

1. Use validacao local na maioria das rotas.
2. Use o gateway em operacoes sensiveis.
3. Sempre use HTTPS em producao.
4. Nunca exponha ou registre o `JWT_SECRET`.
5. Aplique rate limiting quando fizer sentido.
6. Valide entradas antes de usá-las.

---

## Testes

Rode todos os testes:
```bash
go test ./...
```

---

## Troubleshooting

- **"JWT secret not configured"** — chame `authjwt.SetSecret` antes dos helpers/middlewares.
- **"Missing authentication token"** — envie o header `Authorization: Bearer <token>`.
- **"Token expired"** — use `client.RefreshToken` para renovar.
- **"Insufficient permissions"** — verifique a role do usuario.

---

## Documentar endpoints protegidos

- Mantenha uma tabela curta das rotas que usam `RequireAuth`, `RequireAdmin`, `RequireSuperAdmin` ou `RequireRole` para que revisores saibam o que esta protegido.
- Inclua role exigida/escopo de tenant e o servico dono da rota. Exemplo:

| Rota | Metodo | Middleware | Role | Observacao |
| --- | --- | --- | --- | --- |
| `/api/v1/billing/faturas` | GET | `RequireAuth` | qualquer | escopo do tenant |
| `/api/v1/billing/usuarios/:id` | DELETE | `RequireAdmin` | admin | somente admin |
| `/api/v1/system/tenants` | GET | `RequireSuperAdmin` | superadmin | controle da plataforma |

Mantenha essa lista no README do servico ou runbook e atualize quando novas rotas forem adicionadas.

## Configurar CORS

Habilite CORS no servidor API para permitir que os clientes web (console/MFEs) chamem endpoints protegidos com o header `Authorization`.

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

Ajuste `AllowOrigins` por ambiente (dev vs prod) e mantenha `Authorization` em `AllowHeaders` para que o header Bearer flua corretamente.

---

## Links uteis

- Auth Gateway README: `../../services/auth-gateway/README.md`
- Guia Libs vs Services: `../../../docs/architecture/LIBS_VS_SERVICES.md`
- README Platform Auth: `./README-pt-BR.md`
- Exemplo completo: `./examples/basic_usage.go`

---

## Suporte

1. Consulte este guia e os exemplos.
2. Veja a documentacao do auth-gateway.
3. Abra um issue no repositorio se precisar.

---

Ultima atualizacao: Dezembro 2025  
Versao da biblioteca: 1.0.0
