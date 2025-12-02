# Serviço de Gerenciamento de Tenants

Microsserviço de gerenciamento multi-tenant para a plataforma SaaS de Voice of Customer.

## Responsabilidades do Serviço

- Operações CRUD para tenants (empresas)
- Gerenciamento de configuração de tenants (agentes de IA, configurações de telefonia)
- Gerenciamento de chaves de API para autenticação de tenant
- Aplicação de quotas e limites de uso
- Fluxos de onboarding de tenants
- Publicação de eventos para o ciclo de vida de tenants

## Arquitetura

Este serviço segue a **Arquitetura Hexagonal** (Portas e Adaptadores):

```
┌─────────────────────────────────────────────────────────────────┐
│                     ADAPTADORES (Condutores)                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  REST API   │  │   gRPC      │  │   Kafka     │              │
│  │  Handler    │  │   Server    │  │  Consumer   │              │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘              │
│         │                │                │                      │
│         └────────────────┼────────────────┘                      │
│                          ▼                                       │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                   CAMADA DE APLICAÇÃO                      │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │  │
│  │  │   Tenant    │  │   APIKey    │  │   Config    │        │  │
│  │  │   Service   │  │   Service   │  │   Service   │        │  │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │  │
│  └─────────┼────────────────┼────────────────┼────────────────┘  │
│            │                │                │                   │
│            └────────────────┼────────────────┘                   │
│                             ▼                                    │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    CAMADA DE DOMÍNIO                       │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │  │
│  │  │   Tenant    │  │   APIKey    │  │   Config    │        │  │
│  │  │   Entity    │  │   Entity    │  │   Entity    │        │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘        │  │
│  │                                                            │  │
│  │                     PORTAS (Interfaces)                    │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │  │
│  │  │  TenantRepo │  │  APIKeyRepo │  │  EventPub   │        │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘        │  │
│  └───────────────────────────────────────────────────────────┘  │
│                             │                                    │
│                             ▼                                    │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                   ADAPTADORES (Conduzidos)                 │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │  │
│  │  │ PostgreSQL  │  │    Redis    │  │   Kafka     │        │  │
│  │  │   Repo      │  │   Cache     │  │  Publisher  │        │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘        │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Estrutura de Pastas

```
tenant-management/
├── cmd/
│   └── server/
│       └── main.go                 # Ponto de entrada da aplicação
├── internal/
│   ├── domain/                     # Camada de domínio (lógica de negócio)
│   │   ├── tenant/
│   │   │   ├── entity.go           # Entidade de domínio Tenant
│   │   │   ├── repository.go       # Interface do repositório (porta)
│   │   │   ├── service.go          # Serviço de domínio
│   │   │   └── errors.go           # Erros específicos do domínio
│   │   ├── apikey/
│   │   │   ├── entity.go
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   └── events/
│   │       └── events.go           # Eventos de domínio
│   ├── application/                # Camada de aplicação (casos de uso)
│   │   ├── tenant/
│   │   │   ├── service.go          # Serviço de aplicação
│   │   │   ├── dto.go              # DTOs para camada de app
│   │   │   └── commands.go         # Objetos Command/Query
│   │   └── apikey/
│   │       └── service.go
│   ├── adapter/                    # Adaptadores (infraestrutura)
│   │   ├── http/                   # Adaptador HTTP (REST API)
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
│   │   ├── grpc/                   # Adaptador gRPC
│   │   │   ├── server.go
│   │   │   └── handler/
│   │   │       └── tenant.go
│   │   ├── postgres/               # Adaptador PostgreSQL
│   │   │   ├── tenant_repo.go
│   │   │   ├── apikey_repo.go
│   │   │   └── migrations/
│   │   │       ├── 000001_create_tenants.up.sql
│   │   │       ├── 000001_create_tenants.down.sql
│   │   │       └── ...
│   │   ├── redis/                  # Adaptador Redis
│   │   │   └── cache.go
│   │   └── kafka/                  # Adaptador Kafka
│   │       └── publisher.go
│   └── config/                     # Configuração
│       └── config.go
├── pkg/                            # Pacotes compartilhados
│   ├── logger/
│   │   └── logger.go
│   ├── errors/
│   │   └── errors.go
│   ├── pagination/
│   │   └── pagination.go
│   └── middleware/
│       └── correlation.go
├── api/                            # Definições de API
│   ├── openapi/
│   │   └── openapi.yaml
│   └── proto/
│       └── tenant.proto
├── migrations/                     # Migrações de banco de dados
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

## Início Rápido

```bash
# Executar com Docker Compose
docker-compose up -d

# Executar localmente
export $(cat .env | xargs)
go run cmd/server/main.go

# Executar testes
make test

# Gerar documentação OpenAPI
make generate-openapi

# Executar migrações
make migrate-up
```

## Endpoints da API

| Método | Caminho | Descrição |
|--------|---------|-----------|
| POST | /api/v1/tenants | Criar um novo tenant |
| GET | /api/v1/tenants | Listar todos os tenants (admin) |
| GET | /api/v1/tenants/{id} | Obter tenant por ID |
| PUT | /api/v1/tenants/{id} | Atualizar tenant |
| DELETE | /api/v1/tenants/{id} | Excluir tenant (soft delete) |
| GET | /api/v1/tenants/{id}/config | Obter configuração do tenant |
| PUT | /api/v1/tenants/{id}/config | Atualizar configuração do tenant |
| POST | /api/v1/tenants/{id}/api-keys | Criar chave de API |
| GET | /api/v1/tenants/{id}/api-keys | Listar chaves de API |
| DELETE | /api/v1/tenants/{id}/api-keys/{keyId} | Revogar chave de API |
| GET | /health | Verificação de saúde |
| GET | /ready | Verificação de prontidão |
| GET | /metrics | Métricas Prometheus |

## Variáveis de Ambiente

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| SERVER_HOST | Host do servidor | 0.0.0.0 |
| SERVER_PORT | Porta do servidor | 8080 |
| GRPC_PORT | Porta do servidor gRPC | 9090 |
| DATABASE_URL | String de conexão PostgreSQL | - |
| REDIS_URL | String de conexão Redis | - |
| KAFKA_BROKERS | Endereços dos brokers Kafka | - |
| LOG_LEVEL | Nível de logging | info |
| JWT_SECRET | Segredo de assinatura JWT | - |

## Desenvolvimento

### Pré-requisitos

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+
- Kafka 3.0+

### Executando Localmente

1. Clone o repositório
2. Copie `.env.example` para `.env` e configure
3. Inicie as dependências com `docker-compose up -d postgres redis kafka`
4. Execute as migrações com `make migrate-up`
5. Inicie o servidor com `go run cmd/server/main.go`

### Testes

```bash
# Executar todos os testes
make test

# Testes com cobertura
make test-coverage

# Testes de integração
make test-integration
```

## Documentação Relacionada

- [Arquitetura da Plataforma](../../../docs/architecture/README.md)
- [Documentação da API](./docs/api.md)
- [Guia de Deploy](./docs/deployment.md)

---

**Versão**: 1.0.0  
**Licença**: Proprietary
