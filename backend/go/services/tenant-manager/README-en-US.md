# Tenant Management Service

Multi-tenant management microservice for the Voice of Customer SaaS platform.

## Service Responsibilities

- CRUD operations for tenants (companies)
- Tenant configuration management (AI agents, telephony settings)
- API key management for tenant authentication
- Usage quotas and limits enforcement
- Tenant onboarding workflows
- Event publishing for tenant lifecycle events

## Architecture

This service follows **Hexagonal Architecture** (Ports & Adapters):

```
┌───────────────────────────────────────────────────────────┐
│                     ADAPTERS (Driving)                    │
│     ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│     │  REST API   │  │   gRPC      │  │   Kafka     │     │
│     │  Handler    │  │   Server    │  │  Consumer   │     │
│     └──────┬──────┘  └──────┬──────┘  └──────┬──────┘     │
│            │                │                │            │
│            └────────────────┼────────────────┘            │
│                             ▼                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                  APPLICATION LAYER                  │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │  │
│  │  │   Tenant    │  │   APIKey    │  │   Config    │  │  │
│  │  │   Service   │  │   Service   │  │   Service   │  │  │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  │  │
│  └─────────┼────────────────┼────────────────┼─────────┘  │
│            │                │                │            │
│            └────────────────┼────────────────┘            │
│                             ▼                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                    DOMAIN LAYER                     │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │  │
│  │  │   Tenant    │  │   APIKey    │  │   Config    │  │  │
│  │  │   Entity    │  │   Entity    │  │   Entity    │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  │  │
│  │                                                     │  │
│  │                  PORTS (Interfaces)                 │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │  │
│  │  │  TenantRepo │  │  APIKeyRepo │  │  EventPub   │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  │  │
│  └─────────────────────────────────────────────────────┘  │
│                             │                             │
│                             ▼                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                   ADAPTERS (Driven)                 │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │  │
│  │  │ PostgreSQL  │  │    Redis    │  │   Kafka     │  │  │
│  │  │   Repo      │  │   Cache     │  │  Publisher  │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  │  │
│  └─────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────┘
```

## Folder Structure

```
tenant-manager/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── domain/                     # Domain layer (business logic)
│   │   ├── tenant/
│   │   │   ├── entity.go           # Tenant domain entity
│   │   │   ├── repository.go       # Repository interface (port)
│   │   │   ├── service.go          # Domain service
│   │   │   └── errors.go           # Domain-specific errors
│   │   ├── apikey/
│   │   │   ├── entity.go
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   └── events/
│   │       └── events.go           # Domain events
│   ├── application/                # Application layer (use cases)
│   │   ├── tenant/
│   │   │   ├── service.go          # Application service
│   │   │   ├── dto.go              # DTOs for app layer
│   │   │   └── commands.go         # Command/Query objects
│   │   └── apikey/
│   │       └── service.go
│   ├── adapter/                    # Adapters (infrastructure)
│   │   ├── http/                   # HTTP adapter (REST API)
│   │   │   ├── router.go
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go
│   │   │   │   ├── tenant.go
│   │   │   │   ├── logging.go
│   │   │   │   └── recovery.go
│   │   │   ├── handler/
│   │   │   │   ├── tenant.go
│   │   │   │   ├── apikey.go
│   │   │   │   └── health.go
│   │   │   ├── dto/
│   │   │   │   ├── request.go
│   │   │   │   └── response.go
│   │   │   └── validator/
│   │   │       └── validator.go
│   │   ├── grpc/                   # gRPC adapter
│   │   │   ├── server.go
│   │   │   └── handler/
│   │   │       └── tenant.go
│   │   ├── postgres/               # PostgreSQL adapter
│   │   │   ├── tenant_repo.go
│   │   │   ├── apikey_repo.go
│   │   │   └── migrations/
│   │   │       ├── 000001_create_tenants.up.sql
│   │   │       ├── 000001_create_tenants.down.sql
│   │   │       └── ...
│   │   ├── redis/                  # Redis adapter
│   │   │   └── cache.go
│   │   └── kafka/                  # Kafka adapter
│   │       └── publisher.go
│   └── config/                     # Configuration
│       └── config.go
├── pkg/                            # Shared packages
│   ├── logger/
│   │   └── logger.go
│   ├── errors/
│   │   └── errors.go
│   ├── pagination/
│   │   └── pagination.go
│   └── middleware/
│       └── correlation.go
├── api/                            # API definitions
│   ├── openapi/
│   │   └── openapi.yaml
│   └── proto/
│       └── tenant.proto
├── migrations/                     # Database migrations
│   ├── 000001_create_tenants.up.sql
│   └── 000001_create_tenants.down.sql
├── scripts/
│   ├── migrate.sh
│   └── generate.sh
├── Dockerfile
├── Makefile
├── go.mod
└── go.sum
```

## Quick Start

```bash
# Run with Docker Compose
docker-compose up -d

# Run locally
export $(cat .env | xargs)
go run cmd/server/main.go

# Run tests
make test

# Generate OpenAPI docs
make generate-openapi

# Run migrations
make migrate-up
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/tenants | Create a new tenant |
| GET | /api/v1/tenants | List all tenants (admin) |
| GET | /api/v1/tenants/{id} | Get tenant by ID |
| PUT | /api/v1/tenants/{id} | Update tenant |
| DELETE | /api/v1/tenants/{id} | Delete tenant (soft delete) |
| GET | /api/v1/tenants/{id}/config | Get tenant configuration |
| PUT | /api/v1/tenants/{id}/config | Update tenant configuration |
| POST | /api/v1/tenants/{id}/api-keys | Create API key |
| GET | /api/v1/tenants/{id}/api-keys | List API keys |
| DELETE | /api/v1/tenants/{id}/api-keys/{keyId} | Revoke API key |
| GET | /health | Health check |
| GET | /ready | Readiness check |
| GET | /metrics | Prometheus metrics |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| SERVER_HOST | Server host | 0.0.0.0 |
| SERVER_PORT | Server port | 8080 |
| GRPC_PORT | gRPC server port | 9090 |
| DATABASE_URL | PostgreSQL connection string | - |
| REDIS_URL | Redis connection string | - |
| KAFKA_BROKERS | Kafka broker addresses | - |
| LOG_LEVEL | Logging level | info |
| JWT_SECRET | JWT signing secret | - |

## Development

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+
- Kafka 3.0+

### Running Locally

1. Clone the repository
2. Copy `.env.example` to `.env` and configure
3. Start dependencies with `docker-compose up -d postgres redis kafka`
4. Run migrations with `make migrate-up`
5. Start the server with `go run cmd/server/main.go`

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration
```

## Related Documentation

- [Platform Architecture](../../../docs/architecture/README.md)
- [API Documentation](./docs/api.md)
- [Deployment Guide](./docs/deployment.md)

---

**Version**: 1.0.0  
**License**: Proprietary
