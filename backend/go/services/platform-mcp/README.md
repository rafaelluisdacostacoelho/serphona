# platform-mcp

Service scaffold for the Model Context Protocol gateway. Provides HTTP + gRPC entrypoints with platform-auth, health/ready endpoints, Prometheus metrics, and optional pprof.

## Quick start
- `make build` – build linux binary to `bin/`
- `make run` – start the server (HTTP on 8088, gRPC on 9098 by default)
- `make test` – run unit tests
- `make test-coverage` – run tests with coverage report
- `make tidy` – tidy Go modules

## Configuration
Environment variables (see `internal/config/config.go` for defaults):
- `ENVIRONMENT`, `LOG_LEVEL`
- HTTP: `HTTP_HOST`, `HTTP_PORT`, `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_IDLE_TIMEOUT`, `HTTP_MAX_HEADER_BYTES`, `HTTP_SHUTDOWN_TIMEOUT`
- gRPC: `GRPC_HOST`, `GRPC_PORT`, `GRPC_MAX_RECV_MSG_SIZE_MB`, `GRPC_MAX_SEND_MSG_SIZE_MB`, `GRPC_CONNECTION_TIMEOUT`, `GRPC_DEFAULT_REQUEST_TIMEOUT`, `GRPC_REFLECTION_ENABLED`
- Metrics: `METRICS_ENABLED`, `METRICS_PATH`
- pprof: `PPROF_ENABLED`
- Auth: `JWT_SECRET`, `JWT_ALLOWED_ALGS`, `JWT_ISSUER`, `JWT_AUDIENCE`, `JWT_SERVICE_AUDIENCE`, `JWT_CLOCK_SKEW`, `JWT_MAX_TOKEN_BYTES`, `JWKS_URL`, `JWKS_CACHE_TTL`, `JWKS_ALLOWED_KIDS`, `JWT_REQUIRED_SCOPES`, `TENANT_CLAIM`

## Endpoints
- HTTP: `/health`, `/ready`, `/metrics` (when enabled), `/debug/pprof/*` (when enabled), `/api/v1/ping` (auth required)
- gRPC: health service registered; reflection enabled when `GRPC_REFLECTION_ENABLED=true` and `ENVIRONMENT!=production`.
