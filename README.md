# Serphona

**Multi-tenant Voice of Customer SaaS Platform**

Serphona é uma plataforma completa de Voice of Customer (VoC) que combina agentes de voz com IA e analytics avançados para ajudar empresas a entender e melhorar as interações com clientes.

## 🏗️ Arquitetura

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                      SERPHONA PLATFORM                                      │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                             │
│  ┌────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                              FRONTEND (React + TypeScript)                             │ │
│  │                                   frontend/console/                                    │ │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐       │ │
│  │  │  Dashboard  │ │   Agents    │ │    Tools    │ │  Analytics  │ │   Billing   │       │ │
│  │  │    Page     │ │  Config     │ │  & API Keys │ │  Dashboard  │ │   Stripe    │       │ │
│  │  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘       │ │
│  └────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                            │                                                │
│                                            ▼                                                │
│  ┌────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                           BACKEND SERVICES (Kubernetes)                                │ │
│  │                                                                                        │ │
│  │  ┌─────────────────────────── Go Services ───────────────────────────────────────────┐ │ │
│  │  │  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │ │ │
│  │  │  │ auth-gateway    │ │ tenant-manager  │ │ billing-service │ │ analytics-query │  │ │ │
│  │  │  │ (JWT, RBAC)     │ │ (Multi-tenant)  │ │ (Stripe)        │ │ (ClickHouse)    │  │ │ │
│  │  │  └─────────────────┘ └─────────────────┘ └─────────────────┘ └─────────────────┘  │ │ │
│  │  │  ┌─────────────────┐ ┌─────────────────┐                                          │ │ │
│  │  │  │ agent-          │ │ tools-gateway   │   Libs: platform-core, platform-auth,    │ │ │
│  │  │  │ orchestrator    │ │ (MCP, APIs)     │         platform-events, observability   │ │ │
│  │  │  └─────────────────┘ └─────────────────┘                                          │ │ │
│  │  └───────────────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                                                                        │ │
│  │  ┌─────────────────────────── Python Services ───────────────────────────────────────┐ │ │
│  │  │  ┌─────────────────────────┐ ┌─────────────────────────┐                          │ │ │
│  │  │  │ analytics-processor     │ │ reporting-export        │   Libs: analytics-common │ │ │
│  │  │  │ (Kafka → NLP → CH)      │ │ (PDF, CSV exports)      │         nlp-utils        │ │ │
│  │  │  └─────────────────────────┘ └─────────────────────────┘                          │ │ │
│  │  └───────────────────────────────────────────────────────────────────────────────────┘ │ │
│  └────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                            │                                                │
│                                            ▼                                                │
│  ┌────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                                   DATA LAYER                                           │ │
│  │  ┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────┐ │ │
│  │  │  PostgreSQL   │ │  ClickHouse   │ │     Kafka     │ │     MinIO     │ │   Redis   │ │ │
│  │  │ (OLTP + RLS)  │ │   (OLAP)      │ │  (Streaming)  │ │  (S3 Storage) │ │  (Cache)  │ │ │
│  │  └───────────────┘ └───────────────┘ └───────────────┘ └───────────────┘ └───────────┘ │ │
│  └────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                            │                                                │
│                                            ▼                                                │
│  ┌────────────────────────────────────────────────────────────────────────────────────────┐ │
│  │                              VOIP LAYER (Bare Metal)                                   │ │
│  │  ┌───────────────────┐  ┌───────────────────┐  ┌───────────────────┐                   │ │
│  │  │     Asterisk      │  │     Kamailio      │  │    RTPEngine      │                   │ │
│  │  │   (PBX, WebRTC)   │  │   (SIP Proxy)     │  │  (Media Proxy)    │                   │ │
│  │  └───────────────────┘  └───────────────────┘  └───────────────────┘                   │ │
│  └────────────────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

## 📁 Estrutura do Repositório

```
serphona/
├── README.md                           # Este arquivo
├── go.work                             # Go workspace (múltiplos módulos)
├── package.json                        # Scripts unificados (npm/yarn)
├── Makefile                            # Comandos de build
├── docker-compose.dev.yml              # Ambiente dev local
│
├── .github/
│   └── workflows/
│       ├── ci-backend.yml              # CI para Go e Python
│       ├── ci-frontend.yml             # CI para React
│       └── ci-infra.yml                # CI para Terraform/Helm
│
├── infra/                              # Infrastructure as Code
│   ├── terraform/
│   │   ├── envs/
│   │   │   ├── dev/
│   │   │   ├── staging/
│   │   │   └── prod/
│   │   └── modules/
│   │       ├── k8s-cluster/
│   │       ├── postgres/
│   │       ├── kafka/
│   │       ├── clickhouse/
│   │       ├── minio/
│   │       └── monitoring/
│   └── helm/
│       ├── agent-orchestrator/
│       ├── tools-gateway/
│       ├── tenant-manager/
│       ├── analytics-query-service/
│       ├── analytics-processor-service/
│       ├── billing-service/
│       ├── frontend-console/
│       ├── superset/
│       └── metabase/
│
├── backend/
│   ├── go/
│   │   ├── services/
│   │   │   ├── agent-orchestrator/     # Orquestração de agentes IA
│   │   │   ├── tools-gateway/          # Gateway para ferramentas/MCP
│   │   │   ├── tenant-manager/         # Gestão multi-tenant
│   │   │   ├── analytics-query-service/ # Queries ClickHouse
│   │   │   ├── billing-service/        # Integração Stripe
│   │   │   └── auth-gateway/           # Auth JWT/OIDC
│   │   └── libs/
│   │       ├── platform-core/          # Logging, config, db, http
│   │       ├── platform-auth/          # JWT, tenants, RBAC
│   │       ├── platform-events/        # Kafka client
│   │       └── platform-observability/ # Metrics, tracing
│   │
│   └── python/
│       ├── analytics-processor-service/ # Kafka → NLP → ClickHouse
│       ├── reporting-export-service/    # Export PDF, CSV
│       └── libs/
│           ├── analytics-common/        # Shared analytics code
│           └── nlp-utils/               # NLP utilities
│
├── frontend/
│   └── console/                        # React Admin SaaS (multi-tenant)
│       ├── package.json
│       ├── src/
│       │   ├── features/
│       │   ├── components/
│       │   ├── services/
│       │   ├── context/
│       │   ├── routes/
│       │   └── i18n/
│       └── ...
│
└── docs/
    ├── architecture/                   # Diagramas e decisões
    ├── api/                            # OpenAPI specs
    └── decisions/                      # ADRs (Architecture Decision Records)
```

## 🛠️ Tech Stack

### Frontend (`frontend/console/`)
| Tecnologia | Propósito |
|------------|-----------|
| React 18 | UI Framework |
| TypeScript | Type Safety |
| Vite | Build Tool |
| React Router v6 | Routing |
| TanStack Query | Server State |
| i18next | i18n (EN/PT-BR) |
| Tailwind CSS | Styling |
| Recharts | Charts |

### Backend Go (`backend/go/`)
| Tecnologia | Propósito |
|------------|-----------|
| Go 1.21+ | Linguagem |
| Gin | HTTP Framework |
| GORM | ORM PostgreSQL |
| Zap | Logging |
| Viper | Configuration |
| gRPC | Inter-service comm |

### Backend Python (`backend/python/`)
| Tecnologia | Propósito |
|------------|-----------|
| Python 3.11+ | Linguagem |
| FastAPI | HTTP Framework |
| aiokafka | Kafka consumer |
| clickhouse-connect | ClickHouse client |
| Pydantic | Validation |

### Data Layer
| Tecnologia | Propósito |
|------------|-----------|
| PostgreSQL | OLTP + Multi-tenant (RLS) |
| ClickHouse | OLAP Analytics |
| Kafka | Event streaming |
| Redis | Cache, sessions |
| MinIO | Object storage (S3) |

### Infra (`infra/`)
| Tecnologia | Propósito |
|------------|-----------|
| Terraform | IaC |
| Helm | K8s packages |
| Prometheus | Metrics |
| Grafana | Dashboards |
| Loki | Logs |

## 🚀 Quick Start

### Pré-requisitos
- Docker & Docker Compose
- Node.js 18+
- Go 1.21+
- Python 3.11+
- kubectl (para K8s)

### Desenvolvimento Local

```bash
# Clone
git clone https://github.com/your-org/serphona.git
cd serphona

# Suba a infra local (Postgres, Kafka, ClickHouse, Redis, MinIO)
docker-compose -f docker-compose.dev.yml up -d

# Frontend
cd frontend/console
npm install
npm run dev

# Backend Go (tenant-manager)
cd backend/go/services/tenant-manager
go run cmd/server/main.go

# Backend Python (analytics-processor)
cd backend/python/analytics-processor-service
python -m venv venv && source venv/bin/activate
pip install -r requirements.txt
python -m analytics_processor.main
```

### Usando Makefile

```bash
make dev          # Sobe ambiente dev completo
make test         # Roda todos os testes
make build        # Build de todos os serviços
make lint         # Lint Go, Python, TS
```

## 🔐 Segurança

- **TLS Everywhere**: Todo tráfego com TLS 1.3
- **Multi-tenant Isolation**: Row-Level Security (RLS) no PostgreSQL
- **Secrets**: HashiCorp Vault
- **Auth**: JWT + OAuth2/OIDC
- **RBAC**: Por tenant

## 📊 Multi-tenancy

| Camada | Estratégia |
|--------|------------|
| **Database** | RLS com `tenant_id` |
| **Analytics** | ClickHouse particionado por `(tenant_id, month)` |
| **API** | JWT claims incluem `tenant_id` |
| **Kafka** | Eventos taggeados com `tenant_id` |
| **Storage** | MinIO buckets por tenant |

## 📈 Observability

- **Metrics**: Prometheus + Grafana
- **Logs**: Loki + Grafana
- **Traces**: Tempo (OpenTelemetry)
- **Alerts**: Alertmanager

## 📝 Documentação

- [Arquitetura](docs/architecture/)
- [API Specs](docs/api/)
- [ADRs](docs/decisions/)

## 📄 License

Proprietary software. All rights reserved.

---

**Serphona** - Voice of Customer Platform
