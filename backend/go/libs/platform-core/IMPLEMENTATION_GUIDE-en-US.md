# Platform Core - Implementation Guide

Guide to adopt the platform-core library in your services.

## What you get
- Typed `Config` struct covering HTTP/GRPC, database, Redis, Kafka, ClickHouse, MinIO, JWT, OTLP, log level, environment.
- Loader that merges defaults, `config.yaml`, and environment variables (supports comma-separated Kafka brokers).
- Validation helper for required keys.
- Zap logger helper that honors `LOG_LEVEL`.
- Health handler for liveness/readiness endpoints.
- Secrets helper for env-based secrets.

## Setup Steps
1) Add dependency to your service `go.mod`:
```go
require github.com/serphona/serphona/backend/go/libs/platform-core v1.0.0
```
Run `go mod tidy`.

2) Add `config.yaml` (optional) in the service root or `/etc/serphona/`.

3) Define environment variables for secrets/overrides (see list below).

4) Load config on startup:
```go
cfg, err := config.Load()
if err != nil {
    log.Fatalf("config: %v", err)
}

if err := config.ValidateRequired(cfg, "DATABASE_URL", "JWT_SECRET"); err != nil {
    log.Fatalf("missing config: %v", err)
}

logger, err := logger.New(cfg.LogLevel)
if err != nil {
    log.Fatalf("logger: %v", err)
}
defer logger.Sync()

// Health endpoint
mux := http.NewServeMux()
mux.HandleFunc("/health", health.Handler(
    func() error { return nil }, // liveness
    func() error { return nil }, // readiness checks
))
```

5) Use `cfg` across your app (HTTP ports, database URLs, brokers, etc.).

## Environment Keys (defaults)
- `HTTP_ADDR` (`:8080`)
- `GRPC_ADDR` (`:9090`)
- `DATABASE_URL`
- `REDIS_URL`
- `KAFKA_BROKERS` (comma-separated in env is supported)
- `CLICKHOUSE_HOST`
- `CLICKHOUSE_PORT` (`8123`)
- `MINIO_ENDPOINT`
- `MINIO_ACCESS_KEY`
- `MINIO_SECRET_KEY`
- `JWT_SECRET`
- `JWT_EXPIRATION` seconds (`3600`)
- `OTLP_ENDPOINT`
- `LOG_LEVEL` (`info`)
- `ENVIRONMENT` (`development`)

## Sample `config.yaml`
```yaml
HTTP_ADDR: ":8080"
GRPC_ADDR: ":9090"
DATABASE_URL: "postgres://user:pass@localhost:5432/app?sslmode=disable"
REDIS_URL: "redis://localhost:6379"
KAFKA_BROKERS: ["localhost:9092"]
CLICKHOUSE_HOST: "localhost"
CLICKHOUSE_PORT: 8123
MINIO_ENDPOINT: "localhost:9000"
MINIO_ACCESS_KEY: "minio"
MINIO_SECRET_KEY: "minio123"
JWT_SECRET: "change-me"
JWT_EXPIRATION: 3600
OTLP_ENDPOINT: "http://localhost:4317"
LOG_LEVEL: "info"
ENVIRONMENT: "development"
```

## Implementation Checklist
- [ ] Add dependency and run `go mod tidy`
- [ ] Create `config.yaml` (optional) with non-secret defaults
- [ ] Set env vars for secrets (`DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, etc.)
- [ ] Load config at startup with `config.Load()`
- [ ] Validate required keys with `config.ValidateRequired(...)`
- [ ] Initialize zap logger with `logger.New(cfg.LogLevel)`
- [ ] Expose `/health` using `health.Handler(...)`
- [ ] Use `secrets.Get` for env-based secrets (or secret manager) where needed
- [ ] Wire ports and clients using loaded values
- [ ] Document which keys your service requires
- [ ] Add tests that validate required env vars/fields (optional)

## Troubleshooting
- Missing file: loader works without `config.yaml`; rely on env/defaults.
- Missing env: set the variable or provide it via config file.
- Kafka brokers from env: `KAFKA_BROKERS=host1:9092,host2:9092`.

## Support
Internal use only — reach the platform team if you need extra keys or validation helpers.
