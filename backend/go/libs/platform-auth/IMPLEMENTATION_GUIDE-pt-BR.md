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
require github.com/serphona/serphona/backend/go/libs/platform-auth v1.0.0
```

Execute:
```bash
go mod tidy
```

### Passo 2: Configurar variaveis de ambiente

```env
JWT_SECRET=sua-chave-jwt-min-32-caracteres
AUTH_GATEWAY_URL=http://auth-gateway:8080
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

## Checklist de implementacao

- [ ] Adicionar dependencia em `go.mod`
- [ ] Rodar `go mod tidy`
- [ ] Definir `JWT_SECRET` no ambiente (falhar se ausente)
- [ ] Definir `AUTH_GATEWAY_URL` ao usar o cliente HTTP
- [ ] Chamar `authjwt.SetSecret` na inicializacao (uma vez)
- [ ] Inicializar o cliente com `client.New` se precisar validar via gateway
- [ ] Proteger rotas com `RequireAuth`
- [ ] Restringir rotas admin/superadmin com `RequireAdmin`/`RequireSuperAdmin` ou `RequireRole`
- [ ] Ler `userId`/`tenantId` do contexto quando necessario
- [ ] Tratar erros de auth de forma consistente
- [ ] Testar middleware/JWT com tokens validos e invalidos
- [ ] Documentar endpoints protegidos
- [ ] Configurar CORS se necessario

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
