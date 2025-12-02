# Auth Gateway Service

Complete authentication and authorization service for the Serphona platform, with support for OAuth2 (Google, Apple, Microsoft), JWT tokens, and session management.

## 🏗️ Architecture

This service follows **Clean Architecture** with the following layers:

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

## ✨ Features

### Authentication
- ✅ **User registration** with validation
- ✅ **Login** with email and password
- ✅ **JWT Tokens** (Access + Refresh)
- ✅ **Automatic refresh token**
- ✅ **Logout** (session revocation)
- ✅ **Session management** with device tracking

### OAuth 2.0 / Social Login
- ✅ **Google** Sign-In
- ✅ **Microsoft** Sign-In  
- ✅ **Apple** Sign-In with Apple
- ✅ Automatic linking of OAuth accounts to existing users

### Security
- ✅ **Bcrypt** for password hashing
- ✅ **JWT** with refresh tokens
- ✅ Configurable **CORS**
- ✅ **Rate limiting** (to be implemented)
- ✅ **Session tracking** (IP, User-Agent, Device Info)

### Multi-tenancy
- ✅ Native **multi-tenancy** support
- ✅ Data isolation per tenant
- ✅ Automatic tenant creation on registration

## 🚀 How to Run

### Prerequisites

- Go 1.21+
- PostgreSQL 14+
- Docker (optional)

### 1. Configure Environment Variables

```bash
cp .env.example .env
```

Edit the `.env` file with your credentials:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=serphona_auth

# JWT Secret (minimum 32 characters)
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# OAuth (optional)
OAUTH_GOOGLE_ENABLED=true
OAUTH_GOOGLE_CLIENT_ID=your-client-id
OAUTH_GOOGLE_CLIENT_SECRET=your-client-secret
```

### 2. Start PostgreSQL

#### With Docker:
```bash
docker run -d \
  --name serphona-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=serphona_auth \
  -p 5432:5432 \
  postgres:14
```

### 3. Run Migrations

```bash
# Migrations run automatically when starting the server
# Or you can run manually:
psql -h localhost -U postgres -d serphona_auth -f migrations/000001_create_auth_tables.up.sql
```

### 4. Install Dependencies

```bash
go mod download
```

### 5. Run the Server

```bash
go run cmd/server/main.go
```

The server will be available at `http://localhost:8080`

## 📡 API Endpoints

### Public Authentication

#### Register User
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "name": "John Doe",
  "tenantName": "My Company"
}
```

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response:**
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "John Doe",
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

#### Start OAuth Flow
```http
GET /api/v1/auth/oauth/{provider}
```

Available providers: `google`, `microsoft`, `apple`

**Response:**
```json
{
  "url": "https://accounts.google.com/o/oauth2/v2/auth?..."
}
```

#### OAuth Callback (automatic)
```http
GET /api/v1/auth/oauth/{provider}/callback?code=xxx&state=xxx
```

### Protected Routes (Requires Bearer Token)

#### Get Current User
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

## 🔐 Configuring OAuth Providers

### Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google+ API
4. Configure OAuth 2.0:
   - **Authorized redirect URIs**: `http://localhost:8080/api/v1/auth/oauth/google/callback`
5. Copy Client ID and Client Secret to `.env`

### Microsoft OAuth

1. Go to [Azure Portal](https://portal.azure.com/)
2. Register a new application in Azure AD
3. Configure redirect URI: `http://localhost:8080/api/v1/auth/oauth/microsoft/callback`
4. Copy Application (client) ID and Client secret

### Apple Sign In

1. Go to [Apple Developer Portal](https://developer.apple.com/)
2. Configure Sign in with Apple
3. Register Service ID
4. Configure redirect URI: `http://localhost:8080/api/v1/auth/oauth/apple/callback`
5. Generate JWT client secret

## 🗄️ Database Structure

### Table `users`
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

### Table `sessions`
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

### Table `oauth_states`
```sql
- state (VARCHAR, PK)
- provider (VARCHAR)
- redirect_url (TEXT)
- created_at (TIMESTAMP)
- expires_at (TIMESTAMP)
```

## 🧪 Tests

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

## 🔧 Advanced Configuration

### JWT Configuration

```env
JWT_SECRET=your-secret-min-32-chars
JWT_ACCESS_TOKEN_DURATION=15m    # Access token expiry
JWT_REFRESH_TOKEN_DURATION=168h  # Refresh token expiry (7 days)
```

### Database Connection Pool

In `cmd/server/main.go`:
```go
sqlDB.SetMaxIdleConns(10)      # Idle connections
sqlDB.SetMaxOpenConns(100)     # Max open connections
sqlDB.SetConnMaxLifetime(time.Hour)
```

## 📊 Monitoring & Observability

### Health Check
```bash
curl http://localhost:8080/health
```

### Logs
The service uses `zap` for structured logging:
```json
{
  "level": "info",
  "ts": 1234567890.123,
  "msg": "Starting HTTP server",
  "address": "0.0.0.0:8080"
}
```

## 🚨 Troubleshooting

### Error: "failed to connect to database"
- Check if PostgreSQL is running
- Confirm credentials in `.env`
- Test connection: `psql -h localhost -U postgres`

### Error: "invalid token"
- Token expired - use refresh token
- Incorrect JWT_SECRET - check `.env`

### OAuth not working
- Check redirect URIs in provider configurations
- Confirm Client ID and Secret in `.env`
- Use HTTPS in production

## 🔜 Roadmap

- [ ] Rate limiting per IP/user
- [ ] 2FA (Two-Factor Authentication)
- [ ] Password reset via email
- [ ] Email verification
- [ ] Account linking/unlinking
- [ ] Audit logs
- [ ] Redis for session store
- [ ] Refresh token rotation
- [ ] Device management

## 📝 License

Part of the Serphona project. All rights reserved.

## 🤝 Contributing

1. Fork the repository
2. Create a branch for your feature
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## 📧 Support

For support, open an issue in the repository or contact the development team.
