## Serphona — Copilot / AI coding agent instructions

Purpose: provide immediate, actionable context so an AI code agent can be productive in this repo.

- Big picture (one-paragraph): Serphona is a multi-tenant Voice-of-Customer SaaS platform. Frontend is a React/Vite monorepo (`frontend/console`), backend is split between Go services (`backend/go/services` + `backend/go/libs`) and Python processing services (`backend/python/*`). Data layer uses PostgreSQL (RLS), ClickHouse (OLAP), Kafka (streaming), MinIO (S3) and Redis. Infra is managed with Terraform + Helm (`infra/`).

- Key directories to inspect first:
  - `frontend/console` — React app, vite scripts, i18n, TanStack Query.
  - `backend/go/services` — microservices (auth-gateway, tenant-manager, billing-service, agent-orchestrator, tools-gateway, analytics-query-service).
  - `backend/go/libs` — shared Go modules (platform-core, platform-auth, platform-events, platform-observability).
  - `backend/python/analytics-processor-service` — Kafka → NLP → ClickHouse worker (see `src/voc_processor/worker.py`, `kafka_client.py`).
  - `infra/terraform` and `infra/helm` — deployment and infra modules.

- Development workflows & commands (concrete):
  - Local dev stack: `docker-compose -f docker-compose.dev.yml up -d` (starts Postgres, Kafka, ClickHouse, Redis, MinIO).
  - Frontend: `cd frontend/console && npm install && npm run dev`.
  - One Go service (example): `cd backend/go/services/tenant-manager && go run cmd/server/main.go`.
  - Python processor: `cd backend/python/analytics-processor-service && python -m venv venv; venv\Scripts\activate; pip install -r requirements.txt; python -m voc_processor.main`.
  - Unified Makefile targets: `make dev`, `make test`, `make build`, `make lint` (see `Makefile`).
  - CI: check `.github/workflows/ci-backend.yml`, `ci-frontend.yml`, `ci-infra.yml` for lint/test/build steps.

- Conventions and patterns to follow (project-specific):
  - Libs vs services: if code needs its own server/state/deploy → make a service under `backend/go/services`; otherwise put reusable helpers in `backend/go/libs` (see `docs/architecture/LIBS_VS_SERVICES-en-US.md`).
  - Auth & tenant: tenant isolation is pervasive — JWT must include `tenant_id` and DB uses Row-Level Security (RLS). Look for examples in `backend/go/libs/platform-auth` and service middleware usage like `middleware.RequireAuth()`.
  - Events: Kafka messages are tenant-tagged. Python processor expects `tenant_id` on events (see `backend/python/analytics-processor-service/src/voc_processor/models/events.py`).
  - Observability: use the platform-observability libs for metrics/tracing; services expose `/healthz` and Prometheus metrics.

- Integration points and external dependencies to be careful about:
  - Kafka topics and consumer groups (`backend/python/analytics-processor-service` config).
  - ClickHouse table partitioning by `tenant_id` (important for analytic queries and storage costs).
  - Stripe webhooks are handled by `billing-service` — tests or changes must replicate webhook security.
  - Secrets are expected to be in Vault or K8s Secrets in production; do not hardcode credentials.

- Quick examples an agent can use when editing code:
  - Add auth-protected handler in any Go service: import `github.com/serphona/backend/go/libs/platform-auth/middleware` and register `protected.Use(middleware.RequireAuth())` in `cmd/server/main.go`.
  - Update processor batch logic: `backend/python/analytics-processor-service/src/voc_processor/worker.py` — follow BATCH_SIZE & BATCH_FLUSH_SECONDS pattern and preserve offset commit semantics.

- Pull request guidance for AI edits:
  - Keep changes small and focused per service (do not change multiple services/libraries in the same PR unless necessary).
  - Add/modify unit tests where relevant; run `make test` and ensure lint passes (`make lint`).
  - For infra changes, update `infra/terraform/envs/*` and corresponding Helm charts in `infra/helm/`.

- Where to look for more context (files to cite):
  - repo README: `README.md` (high-level architecture & quick start).
  - libs vs services guidance: `docs/architecture/LIBS_VS_SERVICES-en-US.md`.
  - Python processor example: `backend/python/analytics-processor-service/README-en-US.md` and `src/voc_processor/`.
  - CI workflows: `.github/workflows/ci-backend.yml`, `.github/workflows/ci-frontend.yml`.

If anything above is unclear or you need deeper examples (unit tests, API spec snippets, or an onboarding script), tell me which area to expand and I will iterate.  
