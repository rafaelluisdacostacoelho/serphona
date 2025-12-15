# Serphona

**Multi-tenant Voice of Customer SaaS Platform**

Serphona is a complete Voice of Customer (VoC) platform that combines AI-powered voice agents with advanced analytics to help companies understand and improve customer interactions.

## 🏗️ Architecture

```
┌────────────────────────────────────────────────────────────────────────────────────────────┐
│                                     SERPHONA PLATFORM                                      │
├────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                            │
│ ┌────────────────────────────────────────────────────────────────────────────────────────┐ │
│ │                              FRONTEND (React + TypeScript)                             │ │
│ │                                   frontend/console/                                    │ │
│ │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐       │ │
│ │  │  Dashboard  │ │   Agents    │ │    Tools    │ │  Analytics  │ │   Billing   │       │ │
│ │  │    Page     │ │  Config     │ │  & API Keys │ │  Dashboard  │ │   Stripe    │       │ │
│ │  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘       │ │
│ └────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                           │                                                │
│                                           ▼                                                │
│ ┌────────────────────────────────────────────────────────────────────────────────────────┐ │
│ │                              BACKEND SERVICES (Kubernetes)                             │ │
│ │                                                                                        │ │
│ │  ┌───────────────────────────────── Go Services ─────────────────────────────────────┐ │ │
│ │  │  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │ │ │
│ │  │  │ auth-gateway    │ │ tenant-manager  │ │ billing-service │ │ analytics-query │  │ │ │
│ │  │  │ (JWT, RBAC)     │ │ (Multi-tenant)  │ │ (Stripe)        │ │ (ClickHouse)    │  │ │ │
│ │  │  └─────────────────┘ └─────────────────┘ └─────────────────┘ └─────────────────┘  │ │ │
│ │  │  ┌─────────────────┐ ┌─────────────────┐                                          │ │ │
│ │  │  │ agent-          │ │ tools-gateway   │   Libs: platform-core,                   │ │ │
│ │  │  │ orchestrator    │ │ (MCP, APIs)     │         platform-auth,                   │ │ │
│ │  │  └─────────────────┘ └─────────────────┘         platform-events,                 │ │ │
│ │  │                                                  platform-observability           │ │ │
│ │  │                                                                                   │ │ │
│ │  └───────────────────────────────────────────────────────────────────────────────────┘ │ │
│ │                                                                                        │ │
│ │  ┌─────────────────────────────── Python Services ───────────────────────────────────┐ │ │
│ │  │  ┌─────────────────────────┐ ┌─────────────────────────┐                          │ │ │
│ │  │  │ analytics-processor     │ │ reporting-export        │   Libs: analytics-common │ │ │
│ │  │  │ (Kafka → NLP → CH)      │ │ (PDF, CSV exports)      │         nlp-utils        │ │ │
│ │  │  └─────────────────────────┘ └─────────────────────────┘                          │ │ │
│ │  └───────────────────────────────────────────────────────────────────────────────────┘ │ │
│ └────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                           │                                                │
│                                           ▼                                                │
│ ┌────────────────────────────────────────────────────────────────────────────────────────┐ │
│ │                                      DATA LAYER                                        │ │
│ │  ┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────┐ │ │
│ │  │  PostgreSQL   │ │  ClickHouse   │ │     Kafka     │ │     MinIO     │ │   Redis   │ │ │
│ │  │ (OLTP + RLS)  │ │   (OLAP)      │ │  (Streaming)  │ │  (S3 Storage) │ │  (Cache)  │ │ │
│ │  └───────────────┘ └───────────────┘ └───────────────┘ └───────────────┘ └───────────┘ │ │
│ └────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                           │                                                │
│                                           ▼                                                │
│ ┌────────────────────────────────────────────────────────────────────────────────────────┐ │
│ │                                VOIP LAYER (Bare Metal)                                 │ │
│ │          ┌───────────────────┐  ┌───────────────────┐  ┌───────────────────┐           │ │
│ │          │     Asterisk      │  │     Kamailio      │  │    RTPEngine      │           │ │
│ │          │   (PBX, WebRTC)   │  │   (SIP Proxy)     │  │  (Media Proxy)    │           │ │
│ │          └───────────────────┘  └───────────────────┘  └───────────────────┘           │ │
│ └────────────────────────────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────────────────────────┘
```

## 📁 Repository Structure

```
serphona/
├── README.md                           # This file
├── go.work                             # Go workspace (multiple modules)
├── package.json                        # Unified scripts (npm/yarn)
├── Makefile                            # Build commands
├── docker-compose.dev.yml              # Local dev environment
│
├── .github/
│   └── workflows/
│       ├── ci-backend.yml              # CI for Go and Python
│       ├── ci-frontend.yml             # CI for React
│       └── ci-infra.yml                # CI for Terraform/Helm
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
│   │   │   ├── agent-orchestrator/     # AI agent orchestration
│   │   │   ├── tools-gateway/          # Gateway for tools/MCP
│   │   │   ├── tenant-manager/         # Multi-tenant management
│   │   │   ├── analytics-query-service/ # ClickHouse queries
│   │   │   ├── billing-service/        # Stripe integration
│   │   │   └── auth-gateway/           # JWT/OIDC auth
│   │   └── libs/
│   │       ├── platform-core/          # Logging, config, db, http
│   │       ├── platform-auth/          # JWT, tenants, RBAC
│   │       ├── platform-events/        # Kafka client
│   │       └── platform-observability/ # Metrics, tracing
│   │
│   └── python/
│       ├── analytics-processor-service/ # Kafka → NLP → ClickHouse
│       ├── reporting-export-service/    # PDF, CSV exports
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
    ├── architecture/                   # Diagrams and decisions
    ├── api/                            # OpenAPI specs
    └── decisions/                      # ADRs (Architecture Decision Records)
```

## 🛠️ Tech Stack

### Frontend (`frontend/console/`)
| Technology | Purpose |
|------------|---------|
| React 18 | UI Framework |
| TypeScript | Type Safety |
| Vite | Build Tool |
| React Router v6 | Routing |
| TanStack Query | Server State |
| i18next | i18n (EN/PT-BR) |
| Tailwind CSS | Styling |
| Recharts | Charts |

### Backend Go (`backend/go/`)
| Technology | Purpose |
|------------|---------|
| Go 1.24+ | Language |
| Gin | HTTP Framework |
| GORM | PostgreSQL ORM |
| Zap | Logging |
| Viper | Configuration |
| gRPC | Inter-service comm |

### Backend Python (`backend/python/`)
| Technology | Purpose |
|------------|---------|
| Python 3.11+ | Language |
| FastAPI | HTTP Framework |
| aiokafka | Kafka consumer |
| clickhouse-connect | ClickHouse client |
| Pydantic | Validation |

### Data Layer
| Technology | Purpose |
|------------|---------|
| PostgreSQL | OLTP + Multi-tenant (RLS) |
| ClickHouse | OLAP Analytics |
| Kafka | Event streaming |
| Redis | Cache, sessions |
| MinIO | Object storage (S3) |

### Infra (`infra/`)
| Technology | Purpose |
|------------|---------|
| Terraform | IaC |
| Helm | K8s packages |
| Prometheus | Metrics |
| Grafana | Dashboards |
| Loki | Logs |

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Node.js 18+
- Go 1.24+
- Python 3.11+
- kubectl (for K8s)

### Local Development

```bash
# Clone
git clone https://github.com/your-org/serphona.git
cd serphona

# Start local infrastructure (Postgres, Kafka, ClickHouse, Redis, MinIO)
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

### Using Makefile

```bash
make dev          # Start complete dev environment
make test         # Run all tests
make build        # Build all services
make lint         # Lint Go, Python, TS
```

## 🔐 Security

- **TLS Everywhere**: All traffic with TLS 1.3
- **Multi-tenant Isolation**: Row-Level Security (RLS) in PostgreSQL
- **Secrets**: HashiCorp Vault
- **Auth**: JWT + OAuth2/OIDC
- **RBAC**: Per tenant

## 📊 Multi-tenancy

| Layer | Strategy |
|-------|----------|
| **Database** | RLS with `tenant_id` |
| **Analytics** | ClickHouse partitioned by `(tenant_id, month)` |
| **API** | JWT claims include `tenant_id` |
| **Kafka** | Events tagged with `tenant_id` |
| **Storage** | MinIO buckets per tenant |

## 📈 Observability

- **Metrics**: Prometheus + Grafana
- **Logs**: Loki + Grafana
- **Traces**: Tempo (OpenTelemetry)
- **Alerts**: Alertmanager

## 📝 Documentation

- [Architecture](docs/architecture/)
- [API Specs](docs/api/)
- [ADRs](docs/decisions/)

## 📄 License

Proprietary software. All rights reserved.

---

**Serphona** - Voice of Customer Platform
