# Auth Gateway Service

Serviço de autenticação e autorização completo para a plataforma Serphona, com suporte a OAuth2 (Google, Apple, Microsoft), JWT tokens, e gestão de sessões.

## 🏗️ Arquitetura

Este serviço segue **Clean Architecture** com as seguintes camadas:

```
auth-gateway/
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── domain/          # Business entities and interfaces
│   │   └── user/        # User domain entities
│   ├── usecase/         # Business logic
│   │   └── auth/        # Authentication use cases
│   ├── service/         # Domain services
│   │   ├── jwt/         # JWT token management
│   │   └── tenant/      # Tenant management integration
│   ├── adapter/         # External adapters
│   │   ├── http/        # HTTP handlers and middleware
│   │   ├── postgres/    # PostgreSQL repository
│   │   └── oauth/       # OAuth providers (Google, Microsoft, Apple)
│   └── config/          # Configuration management
└── migrations/          # Database migrations
```

## ✨ Funcionalidades

### Autenticação
- ✅ **Registro de usuários** com validação
- ✅ **Login** com email e senha
- ✅ **JWT Tokens** (Access + Refresh)
- ✅ **Refresh Token** automático
- ✅ **Logout** (revogação de sessões)
- ✅ **Gestão de sessões** com tracking de dispositivos

### OAuth 2.0 / Social Login
- ✅ **Google** Sign-In
- ✅ **Microsoft** Sign-In  
- ✅ **Apple** Sign-In with Apple
- ✅ Vinculação automática de contas OAuth a usuários existentes

### Segurança
- ✅ **Bcrypt** para hash de senhas
- ✅ **JWT** com refresh tokens
- ✅ **CORS** configurável
- ✅ **Rate limiting** (a implementar)
- ✅ **Session tracking** (IP, User-Agent, Device Info)

### Multi-tenancy
- ✅ Suporte a **multi-tenancy** nativo
- ✅ Isolamento de dados por tenant
- ✅ Criação automática de tenant no registro

## 🚀 Como Executar

### Pré-requisitos

- Go 1.21+
- PostgreSQL 14+
- Docker (opcional)

### 1. Configurar Variáveis de Ambiente

```bash
cp .env.example .env
```

Edite o arquivo `.env` com suas credenciais:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=serphona_auth

# JWT Secret (mínimo 32 caracteres)
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# OAuth (opcional)
OAUTH_GOOGLE_ENABLED=true
OAUTH_GOOGLE_CLIENT_ID=your-client-id
OAUTH_GOOGLE_CLIENT_SECRET=your-client-secret
```

### 2. Iniciar PostgreSQL

#### Com Docker:
```bash
docker run -d \
  --name serphona-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=serphona_auth \
  -p 5432:5432 \
  postgres:14
```

### 3. Executar Migrations

```bash
# As migrations são executadas automaticamente ao iniciar o servidor
# Ou você pode rodar manualmente:
psql -h localhost -U postgres -d serphona_auth -f migrations/000001_create_auth_tables.up.sql
```

### 4. Instalar Dependências

```bash
go mod download
```

### 5. Executar o Servidor

```bash
go run cmd/server/main.go
```

O servidor estará disponível em `http://localhost:8080`

## 📡 API Endpoints

### Autenticação Pública

#### Registrar Usuário
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "senha123",
  "name": "João Silva",
  "tenantName": "Minha Empresa"
}
```

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "senha123"
}
```

**Response:**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "João Silva",
    "role": "user",
    "tenantId": "uuid"
  },
  "tokens": {
    "accessToken": "eyJhbGc...",
    "refreshToken": "eyJhbGc...",
    "expiresIn": 900
  }
}
```

#### Refresh Token
```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refreshToken": "eyJhbGc..."
}
```

### OAuth 2.0

#### Iniciar OAuth Flow
```http
GET /api/v1/auth/oauth/{provider}
```

Providers disponíveis: `google`, `microsoft`, `apple`

**Response:**
```json
{
  "url": "https://accounts.google.com/o/oauth2/v2/auth?..."
}
```

#### Callback OAuth (automático)
```http
GET /api/v1/auth/oauth/{provider}/callback?code=xxx&state=xxx
```

### Rotas Protegidas (Requer Bearer Token)

#### Obter Usuário Atual
```http
GET /api/v1/auth/me
Authorization: Bearer {accessToken}
```

#### Logout
```http
POST /api/v1/auth/logout
Authorization: Bearer {accessToken}
```

### Health Check
```http
GET /health
```

## 🔐 Configurando OAuth Providers

### Google OAuth

1. Acesse [Google Cloud Console](https://console.cloud.google.com/)
2. Crie um novo projeto ou selecione um existente
3. Ative a Google+ API
4. Configure OAuth 2.0:
   - **Authorized redirect URIs**: `http://localhost:8080/api/v1/auth/oauth/google/callback`
5. Copie Client ID e Client Secret para o `.env`

### Microsoft OAuth

1. Acesse [Azure Portal](https://portal.azure.com/)
2. Registre uma nova aplicação em Azure AD
3. Configure redirect URI: `http://localhost:8080/api/v1/auth/oauth/microsoft/callback`
4. Copie Application (client) ID e Client secret

### Apple Sign In

1. Acesse [Apple Developer Portal](https://developer.apple.com/)
2. Configure Sign in with Apple
3. Registre Service ID
4. Configure redirect URI: `http://localhost:8080/api/v1/auth/oauth/apple/callback`
5. Gere JWT client secret

## 🗄️ Estrutura do Banco de Dados

### Tabela `users`
```sql
- id (UUID, PK)
- email (VARCHAR, UNIQUE)
- password (VARCHAR)
- name (VARCHAR)
- role (VARCHAR)
- tenant_id (UUID)
- provider (VARCHAR) - 'local', 'google', 'microsoft', 'apple'
- provider_id (VARCHAR)
- verified (BOOLEAN)
- active (BOOLEAN)
- created_at (TIMESTAMP)
- updated_at (TIMESTAMP)
- deleted_at (TIMESTAMP)
```

### Tabela `sessions`
```sql
- id (UUID, PK)
- user_id (UUID, FK)
- refresh_token (TEXT, UNIQUE)
- device_info (TEXT)
- ip_address (VARCHAR)
- user_agent (TEXT)
- expires_at (TIMESTAMP)
- created_at (TIMESTAMP)
- revoked_at (TIMESTAMP)
```

### Tabela `oauth_states`
```sql
- state (VARCHAR, PK)
- provider (VARCHAR)
- redirect_url (TEXT)
- created_at (TIMESTAMP)
- expires_at (TIMESTAMP)
```

## 🧪 Testes

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...
```

## 🐳 Docker

```dockerfile
# Build
docker build -t serphona-auth-gateway .

# Run
docker run -p 8080:8080 \
  -e DB_HOST=host.docker.internal \
  -e DB_PASSWORD=postgres \
  serphona-auth-gateway
```

## 🔧 Configuração Avançada

### JWT Configuration

```env
JWT_SECRET=your-secret-min-32-chars
JWT_ACCESS_TOKEN_DURATION=15m    # Access token expiry
JWT_REFRESH_TOKEN_DURATION=168h  # Refresh token expiry (7 days)
```

### Database Connection Pool

No código `cmd/server/main.go`:
```go
sqlDB.SetMaxIdleConns(10)      # Connections idle
sqlDB.SetMaxOpenConns(100)     # Max open connections
sqlDB.SetConnMaxLifetime(time.Hour)
```

## 📊 Monitoring & Observability

### Health Check
```bash
curl http://localhost:8080/health
```

### Logs
O serviço usa `zap` para logging estruturado:
```json
{
  "level": "info",
  "ts": 1234567890.123,
  "msg": "Starting HTTP server",
  "address": "0.0.0.0:8080"
}
```

## 🚨 Troubleshooting

### Erro: "failed to connect to database"
- Verifique se PostgreSQL está rodando
- Confirme as credenciais no `.env`
- Teste conexão: `psql -h localhost -U postgres`

### Erro: "invalid token"
- Token expirado - use refresh token
- JWT_SECRET incorreto - verifique `.env`

### OAuth não funciona
- Verifique redirect URIs nas configurações dos providers
- Confirme Client ID e Secret no `.env`
- Use HTTPS em produção

## 🔜 Roadmap

- [ ] Rate limiting por IP/usuário
- [ ] 2FA (Two-Factor Authentication)
- [ ] Password reset via email
- [ ] Email verification
- [ ] Account linking/unlinking
- [ ] Audit logs
- [ ] Redis para session store
- [ ] Refresh token rotation
- [ ] Device management

## 📝 License

Parte do projeto Serphona. Todos os direitos reservados.

## 🤝 Contribuindo

1. Fork o repositório
2. Crie uma branch para sua feature
3. Commit suas mudanças
4. Push para a branch
5. Abra um Pull Request

## 📧 Suporte

Para suporte, abra uma issue no repositório ou entre em contato com a equipe de desenvolvimento.
