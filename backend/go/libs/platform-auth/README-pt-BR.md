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

Chamadas serviço-a-serviço podem usar um token de serviço (client credentials) sem sobrescrever tokens de usuário:

```go
svcToken := os.Getenv("SERVICE_AUTH_TOKEN")
authClient := client.NewWithOptions(
    "http://auth-gateway:8080",
    client.WithStaticBearerToken(svcToken),
    client.WithServiceIdentity("billing-service", "billing-service-1"),
)

// Se o handler nao setar Authorization, o bearer estatico entra; tokens de usuario sao preservados.
resp, err := authClient.ValidateToken("user-token")
```

Tokens de servico precisam ser emitidos com claims `service` e `scopes` e usar a audience de servico:

```go
serviceAudience := os.Getenv("SERVICE_AUDIENCE")
authjwt.SetValidationConfig(authjwt.ValidationConfig{
    Audience:    serviceAudience,      // deve casar com a SERVICE_AUDIENCE na emissao
    AllowedAlgs: []string{"RS256"},   // preferir chaves assimetricas para chamadas internas
    AllowedKIDs: []string{"kid-1"},   // allow-list opcional
})
```

O emissor deve preencher `service` (ID do chamador), `tenantId` (`platform` permitido para infra) e `scopes` nao vazios por acao interna.

Tambem e possivel instanciar um cliente endurecido a partir das variaveis de ambiente (URL + identidade + bearer estatico opcional):

```go
svcClient, err := client.NewServiceClientFromEnv("billing-service", "billing-1")
if err != nil {
    log.Fatal(err)
}
```

### 4. Validacao JWT manual

```go
claims, err := authjwt.ValidateToken(tokenString)
rawToken, err := authjwt.ExtractTokenFromHeader(authHeader)
```

### Envelopes de resposta (Gin / chi / gRPC)

Exemplo Gin (adiciona `trace_id` e `request_id` quando `RequireAuth` ja rodou):

```go
func listarFaturas(c *gin.Context) {
    data := []gin.H{{"id": "inv-1", "status": "pago"}}
    response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, data, response.WithPagination(response.Pagination{Page: 1, PageSize: 10, Total: 1, TotalPages: 1}))
}

func criarFatura(c *gin.Context) {
    response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_INPUT", "payload invalido", nil)
}
```

Exemplo net/http (compatível com chi):

```go
mux := http.NewServeMux()
mux.Handle("/reports", middleware.RequireAuthHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    payload := []map[string]any{{"id": "r1"}}
    response.WriteSuccess(r.Context(), w, http.StatusOK, payload)
})))
```

gRPC: use os interceptors para carregar request/trace ID em metadata. Erros mapeiam para `codes.Unauthenticated`/`PermissionDenied`/`Internal` com `errdetails.ErrorInfo` preenchido a partir do codigo de auth:

```go
grpcServer := grpc.NewServer(
    grpc.UnaryInterceptor(middleware.UnaryAuthInterceptor("scope:read")),
)
```

Exemplos executaveis dos envelopes estao em `examples/envelope_gin` e `examples/envelope_http`.

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
SERVICE_AUDIENCE=serphona-service         # audience para tokens de servico
SERVICE_AUTH_TOKEN=internal-service-token # bearer estatico opcional para chamadas internas
SERVICE_NAME=billing-service              # identidade opcional quando usar NewServiceClientFromEnv
SERVICE_INSTANCE=billing-1                # instancia opcional quando usar NewServiceClientFromEnv
TENANT_ID_HEADER=X-Tenant-Id              # opcional; padrao segue a constante da biblioteca
TRACE_REQUEST_HEADER=X-Request-Id         # opcional para servicos
TLS_CA=/etc/ssl/certs/ca.pem              # CA opcional para chamadas ao auth-gateway
TLS_CERT=/etc/ssl/certs/client.pem        # cert opcional para mTLS
TLS_KEY=/etc/ssl/private/client.key       # chave opcional para mTLS
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
- `middleware.TenantIDFromContext(ctx)` — resolve o tenant a partir dos claims ou contexto explicito
- `middleware.EnsureTenantHeader(headers, tenantID)` — injeta o tenant em headers de saida sem mutar o caller

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

### Uso dos helpers de Tenant/RLS (DB, Kafka, HTTP)

```go
// Resolver tenant dos claims e aplicar antes de gravar em DB/Kafka
tenantID, err := middleware.TenantIDFromContext(ctx)
if err != nil {
    return err
}
if err := middleware.EnforceTenant(ctx, tenantIDPayload); err != nil {
    return err // bloqueia cross-tenant
}

// Propagar tenant em Kafka/HTTP
headers := middleware.EnsureTenantHeader(nil, tenantID)
msg.Headers = append(msg.Headers, kafka.Header{Key: middleware.TenantIDHeader, Value: []byte(tenantID)})
req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
req.Header = middleware.EnsureTenantHeader(req.Header, tenantID)
```

Os helpers preferem o tenant presente nos claims; `WithTenantID` pode ser usado para definir tenant explicitamente em fluxos sistema-a-sistema.

## Contribuicao

1. Crie uma branch
2. Realize as mudancas
3. Adicione testes
4. Abra um Pull Request

Documentacao adicional: `IMPLEMENTATION_GUIDE-en-US.md` e `IMPLEMENTATION_GUIDE-pt-BR.md`.

---

Versao: 1.0.0  
Licenca: Proprietaria
