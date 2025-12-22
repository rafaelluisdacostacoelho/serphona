# Biblioteca Platform Auth

Utilitarios de autenticacao compartilhados para os microsservicos da Serphona (o auth-gateway concentra a logica completa de autenticacao).

## Objetivo

Esta biblioteca **nao** implementa:
- Login/Logout
- Registro de usuarios ou gestao de base de dados
- Provedores OAuth
- Emissao de tokens

Esta biblioteca **fornece**:
- Middleware de validacao JWT (Gin)
- Cliente HTTP para falar com o auth-gateway
- Tipos compartilhados (Claims, User, tokens)
- Utilitarios JWT
- Erros padronizados

## Instalacao

```bash
go get github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth
```

## Uso

### 1. Middleware de autenticacao

```go
router := gin.Default()

router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

protegidas := router.Group("/api/v1")
protegidas.Use(middleware.RequireAuth())
{
    protegidas.GET("/faturas", listarFaturas)
    protegidas.GET("/tenants/atual", tenantAtual)
}
```

### 2. Extrair informacoes do usuario

```go
claims, err := middleware.GetClaimsFromContext(c)
if err != nil {
    c.JSON(401, gin.H{"error": "Unauthorized"})
    return
}

log.Printf("Tenant: %s", claims.TenantID)
```

### 3. Cliente HTTP para o auth-gateway

```go
authClient := client.New("http://auth-gateway:8080")
claims, err := authClient.ValidateToken(token)
usuario, err := authClient.GetUserByID(userID, token)
```

### 4. Validacao JWT manual

```go
claims, err := authjwt.ValidateToken(tokenString)
rawToken, err := authjwt.ExtractTokenFromHeader(authHeader)
```

## Estrutura

```
platform-auth/
├── middleware/          # RequireAuth e middlewares de role
├── client/              # Cliente HTTP para auth-gateway
├── jwt/                 # Helpers JWT (validacao/extracao)
├── types/               # Claims, User e tipos de token
├── errors/              # Erros padronizados e codigos
├── examples/            # Exemplo basico com Gin
├── README*.md
├── IMPLEMENTATION_GUIDE*.md
├── go.mod
└── go.sum
```

## Configuracao

Variaveis de ambiente esperadas pelos consumidores:

```env
JWT_SECRET=sua-chave-jwt-super-secreta
AUTH_GATEWAY_URL=http://auth-gateway:8080
```

Configure o secret no inicio da aplicacao (falha se nao estiver setado):

```go
authjwt.MustSetSecretFromEnv()
```

## API

- `middleware.RequireAuth()` — valida o JWT e injeta claims no contexto
- `middleware.RequireRole(role)` — exige uma role especifica
- `middleware.RequireAdmin()` / `RequireSuperAdmin()` — atalhos de roles
- `middleware.GetClaimsFromContext(c)` — retorna `*types.Claims`
- `client.New(baseURL)` — cria cliente para auth-gateway
- `client.ValidateToken(token)` — valida via gateway
- `client.GetUserByID(userID, token)` / `client.GetMe(token)`
- `client.RefreshToken(refreshToken)` / `client.Logout(token)`
- `jwt.ValidateToken(token)` — validacao local com secret configurado
- `jwt.ExtractTokenFromHeader(header)` — parse de `Authorization: Bearer <token>`

## Seguranca

- Valide tokens localmente para rotas comuns; use o auth-gateway em fluxos sensiveis.
- Mantenha o `JWT_SECRET` igual em todos os servicos e nunca o registre em logs.
- Use HTTPS em producao.

## Testes

```bash
go test ./...
```

## Exemplos

Veja `examples/` para um exemplo executavel com Gin.

## Contribuicao

1. Crie uma branch
2. Realize as mudancas
3. Adicione testes
4. Abra um Pull Request

Documentacao adicional: `IMPLEMENTATION_GUIDE-en-US.md` e `IMPLEMENTATION_GUIDE-pt-BR.md`.

---

Versao: 1.0.0  
Licenca: Proprietaria
