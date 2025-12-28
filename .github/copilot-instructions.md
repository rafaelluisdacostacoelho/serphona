## Serphona — Copilot / AI coding agent instructions

Purpose: concise, actionable context so an AI agent can contribute safely.

- Big picture: multi-tenant Voice-of-Customer platform. Frontend: React/Vite console plus auth/billing MFEs and marketing site under frontend/. Backend: Go microservices in backend/go/services (auth-gateway, tenant-manager, billing-service, agent-orchestrator, tools-gateway, analytics-query-service, voice-gateway) with shared libs in backend/go/libs (platform-core/auth/events/observability). Python side at backend/python/services with analytics-processor-service handling Kafka → NLP → ClickHouse. Data layer: PostgreSQL with RLS, ClickHouse for OLAP, Kafka, MinIO, Redis; telephony stack uses Asterisk/Kamailio/RTPEngine (see root README diagram).

- Key docs: architecture index in docs/architecture/README.md; RAG/MCP flows in docs/architecture/RAG-MCP-en-US.md; prompt schema in docs/architecture/PROMPTS-YAML-SPEC-en-US.md; telephony/voice gateway in docs/architecture/VOICE-GATEWAY-DESIGN-en-US.md and TENANT-MANAGER-TELEPHONY-EXTENSIONS-en-US.md; tools gateway in docs/architecture/TOOLS-GATEWAY-ARCHITECTURE-en-US.md; libs vs services guidance in docs/architecture/LIBS_VS_SERVICES-en-US.md; service-specific READMEs under backend/go/services/* and backend/python/services/analytics-processor-service/README.md.

- Local workflows: docker-compose -f docker-compose.yml up -d for full stack; docker-compose -f docker-compose.tests.yml up -d when you only need Kafka for tests. Makefile targets: make dev/test/build/lint. Frontend dev: cd frontend/console|auth-mfe|billing-mfe|website, npm install, npm run dev. Go service example: cd backend/go/services/tenant-manager && go run cmd/server/main.go. Python analytics processor: cd backend/python/services/analytics-processor-service && python -m venv venv && source venv/bin/activate && pip install -r requirements.txt && python -m voc_processor.main.

- Patterns and contracts: multi-tenancy everywhere (JWT claim tenant_id, Postgres RLS, ClickHouse partitions, Kafka events carry tenant_id). Go routing uses platform-auth middleware.RequireAuth(). Observability via platform-observability; expose /healthz and Prometheus metrics. Kafka/ClickHouse configs live in the processor’s config.py; batching is governed by worker.py constants BATCH_SIZE/BATCH_FLUSH_SECONDS and offsets commit after successful batch.

- Testing expectations: Go uses table-driven tests; run go test -cover ./... for units; integration/e2e live under test/integration or test/e2e with build tags -tags=integration|-tags=e2e and helper runners backend/go/services/test/integration/run-tests.(sh|bat) or backend/go/libs/test/integration/run-tests.(sh|bat) which may spin up docker-compose.tests.yml when --with-compose. Python: pytest --cov=src --cov-report=xml in analytics-processor-service; prefer fakes/mocks for Kafka/ClickHouse/Redis. Frontend is React/Vite; follow per-package scripts (npm run test/build).

- Infra: Terraform/Helm under infra/ (envs under infra/terraform/envs/*, charts under infra/helm/). Keep env-specific Helm/TF values in sync with service changes. CI entrypoints are .github/workflows/ci-backend.yml, ci-frontend.yml, ci-infra.yml.

- External/secure handling: never hardcode secrets (Vault/K8s secrets expected). Stripe webhooks in billing-service require signature verification in tests. Kafka topic/consumer group choices are part of contract with analytics-processor; keep tenant_id tagging intact. Telephony components rely on network zoning (see infra docs) if you touch SIP/voice paths.

- Localization rule: when adding docs, provide both en-US and pt-BR variants (name *-en-US.md and *-pt-BR.md).

- If unsure about patterns or new surface area, point to the relevant doc/README above and ask for confirmation before large refactors.