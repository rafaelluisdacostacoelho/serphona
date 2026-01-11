# Architecture Documentation

This directory contains architecture documentation, diagrams, and design documents.

## Contents

- System architecture diagrams
- Component interaction flows
- Data flow diagrams
- Infrastructure topology

## Test & Coverage Standard

- Go services/libs: `gofmt -l .` then `go test -cover ./...` (integration: `go test -tags=integration ./...` with docker-compose deps). Upload `coverage.out` to Codecov per service/lib.
- Python services: `pytest -v --cov=src --cov-report=xml` (unit only; stub Kafka/ClickHouse/Redis). CI uploads `coverage.xml` per service.
- Frontend (console/auth-mfe/billing-mfe/website): `npm run lint`, `npx tsc --noEmit`, tests with `npm run test:coverage` (or `npm run test -- --coverage` where defined). CI uploads `coverage/lcov.info` when present.

## See Also

- [API Documentation](../api/)
- [Architecture Decision Records](../decisions/)
- [Architecture Book Index](BOOK-INDEX-en-US.md)
- [Architecture Diagrams](DIAGRAMS-en-US.md)
