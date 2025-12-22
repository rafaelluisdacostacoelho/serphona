## Serphona — Copilot / AI coding agent instructions

Purpose: provide immediate, actionable context so an AI code agent can be productive in this repo.

- Big picture: Serphona is a multi-tenant Voice-of-Customer SaaS platform. Frontend targets React (console + MFEs + marketing site). Backend is split between Go services (`backend/go/services` + `backend/go/libs`) and Python processing services (`backend/python/*`). Data layer uses PostgreSQL (RLS), ClickHouse (OLAP), Kafka (streaming), MinIO (S3), and Redis. Infra is managed with Terraform + Helm (`infra/`).

- Key directories to inspect first:
  - `frontend/console`, `frontend/auth-mfe`, `frontend/billing-mfe` — React console and MFEs; `frontend/website` for the marketing site.
  - `backend/go/services` — microservices (auth-gateway, tenant-manager, billing-service, agent-orchestrator, tools-gateway, analytics-query-service, voice-gateway).
  - `backend/go/libs` — shared Go modules (platform-core, platform-auth, platform-events, platform-observability).
  - `backend/python/analytics-processor-service` — Kafka → NLP → ClickHouse worker (`src/voc_processor/worker.py`, `kafka_client.py`).
  - `infra/terraform` and `infra/helm` — deployment and infra modules.

- Development workflows & commands:
  - Local stack: `docker-compose -f docker-compose.yml up -d` (Postgres, Redis, Kafka, services); tests-only stack: `docker-compose -f docker-compose.tests.yml up -d` when you just need Kafka.
  - Frontend (React): `cd frontend/console && npm install && npm run dev`; MFEs under `frontend/auth-mfe` and `frontend/billing-mfe` follow the same pattern; website: `cd frontend/website && npm install && npm run dev`.
  - Go service example: `cd backend/go/services/tenant-manager && go run cmd/server/main.go`.
  - Python processor: `cd backend/python/analytics-processor-service && python -m venv venv; venv\Scripts\activate; pip install -r requirements.txt; python -m voc_processor.main`.
  - Make targets: `make dev`, `make test`, `make build`, `make lint` (see `Makefile`).
  - CI entrypoints: `.github/workflows/ci-backend.yml`, `ci-frontend.yml`, `ci-infra.yml`.

- Conventions and patterns:
  - Libs vs services: server/state/deploy → `backend/go/services`; reusable helpers → `backend/go/libs` (see `docs/architecture/LIBS_VS_SERVICES-en-US.md`).
  - Auth/tenant: JWT must carry `tenant_id`; DB uses RLS. See `backend/go/libs/platform-auth` and service middleware `middleware.RequireAuth()`.
  - Events: Kafka messages include `tenant_id`. Processor expects it (see `backend/python/analytics-processor-service/src/voc_processor/models/events.py`).
  - Observability: use platform-observability libs; services expose `/healthz` and Prometheus metrics.

- External dependencies to watch:
  - Kafka topics and consumer groups (processor config).
  - ClickHouse partitioning by `tenant_id`.
  - Stripe webhooks in billing-service: replicate webhook security in tests.
  - Secrets come from Vault/K8s Secrets; never hardcode credentials.

- Quick examples:
  - Auth-protected handler in Go: import `github.com/serphona/backend/go/libs/platform-auth/middleware`; register `protected.Use(middleware.RequireAuth())` in `cmd/server/main.go`.
  - Processor batch logic: `backend/python/analytics-processor-service/src/voc_processor/worker.py` — keep BATCH_SIZE/BATCH_FLUSH_SECONDS and offset commit semantics.

- Testing & coverage patterns:
  - Go: table-driven tests; inject fakes/stubs (see platform-events writer/reader fakes). `go test -cover ./...` for unit; `-tags=integration` and `-tags=e2e` with docker-compose for heavier suites. Gate gofmt (`gofmt -l`) and golangci-lint in CI.
  - Go integration/e2e: place suites under `test/integration` and `test/e2e` with matching build tags; ship helper scripts + README inside `test/integration`. Services use `backend/go/services/test/integration/run-tests.(sh|bat)` (legacy `run-integration-tests.*` should delegate) and libs use `backend/go/libs/test/integration/run-tests.(sh|bat)`. Helpers may start `docker-compose.tests.yml` when `--with-compose` and must be added whenever introducing integration/e2e tests (mirror platform-events pattern, including Kafka topic setup when needed).
  - Python: pytest with fakes/mocks for Kafka/ClickHouse/Redis; avoid real services. `pytest --cov=src --cov-report=xml` in CI.
  - Frontend (Angular/Fuse): Angular Testing Library/Jasmine/Karma; stub HTTP with `HttpTestingController`; avoid real network. `ng test --watch=false --code-coverage` (or template equivalent); consider Cypress e2e separately.
  - Integration tests: place under `test/integration` with wrapper script (e.g., `run-integration-tests`), optional `--with-compose` to start deps.
  - Coverage upload: prefer Codecov/artifacts per job.

- Documentation localization:
  - Always produce docs in both Portuguese and English (`*-pt-BR.md` and `*-en-US.md`).

- PR guidance:
  - Keep changes small and focused per service/library.
  - Add/adjust unit tests where relevant; run `make test` and `make lint`.
  - Infra changes: update `infra/terraform/envs/*` and matching Helm charts under `infra/helm/`.

- Where to look for more context:
  - Root `README.md` (architecture & quick start).
  - `docs/architecture/LIBS_VS_SERVICES-en-US.md` (libs vs services guidance).
  - `backend/python/analytics-processor-service/README-en-US.md` and `src/voc_processor/`.
  - CI workflows: `.github/workflows/ci-backend.yml`, `.github/workflows/ci-frontend.yml`.

If you need deeper examples (unit tests, API snippets, onboarding scripts), ask which area to expand and iterate.
