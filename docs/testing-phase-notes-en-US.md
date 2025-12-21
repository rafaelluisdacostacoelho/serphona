# Testing Phase Notes

## Completed
- Analytics Processor: offline unit tests with in-memory fakes for Kafka, ClickHouse, and Redis (`tests/fakes.py`), exercising `ConsumerWorker` without external services (`pytest -v --cov=src --cov-report=xml`).
- Reporting Export Service: FastAPI smoke tests for health, export lifecycle, and delivery endpoints using `TestClient`, no ClickHouse/MinIO network access required.

## Next Steps
- Frontend (console, auth-mfe, billing-mfe, website): add Vitest + React Testing Library + MSW harness, default `test`/`test:coverage` scripts, and sample HTTP-mocked tests to prevent real calls.
- Document the new frontend test commands in each README and align CI coverage upload once harnesses are in place.
- Close the backlog item by syncing README/notes after frontend harnesses are merged.
