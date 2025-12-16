# Platform Core Library

Core utilities shared across Serphona services. Currently provides configuration loading with sane defaults using Viper.

## Features
- Centralized config loading via YAML + environment variables.
- Defaults for common settings (HTTP/GRPC ports, log level, JWT expiration, ClickHouse port).
- Typed `Config` struct for services to depend on.
- Validation helper to ensure required keys are set.
- Zap logger helper that honors the configured log level.
- Health handler helper for liveness/readiness.
- Simple secrets helper for env-based secrets.

## Installation
```bash
go get github.com/serphona/serphona/backend/go/libs/platform-core
```

## Quickstart
```go
package main

import (
    "log"

    "github.com/serphona/serphona/backend/go/libs/platform-core/config"
)

func main() {
cfg, err := config.Load()
if err != nil {
    log.Fatalf("config: %v", err)
}

log.Printf("HTTP listening on %s, env: %s, log level: %s", cfg.HTTPAddr, cfg.Environment, cfg.LogLevel)

if err := config.ValidateRequired(cfg, "DATABASE_URL", "JWT_SECRET"); err != nil {
    log.Fatalf("missing required config: %v", err)
}

logger, err := logger.New(cfg.LogLevel)
if err != nil {
    log.Fatalf("logger: %v", err)
}
defer logger.Sync()

logger.Info("service starting", zap.String("env", cfg.Environment))
}
```

## Health checks
```go
mux := http.NewServeMux()
mux.HandleFunc("/health", health.Handler(
    func() error { return nil },          // liveness
    func() error { return nil },          // readiness checks (db, cache, etc.)
))
```

## Secrets
```go
dbPass, err := secrets.Get("DB_PASSWORD")
if err != nil {
    log.Fatal(err)
}
```

## Configuration
The loader reads (in order):
1. Defaults (see below)
2. `config.yaml` (cwd or `/etc/serphona/`)
3. Environment variables (override file/defaults)

### Supported keys
- `HTTP_ADDR` (default `:8080`)
- `GRPC_ADDR` (default `:9090`)
- `DATABASE_URL`
- `REDIS_URL`
- `KAFKA_BROKERS` (comma-separated env is accepted by Viper)
- `CLICKHOUSE_HOST`
- `CLICKHOUSE_PORT` (default `8123`)
- `MINIO_ENDPOINT`
- `MINIO_ACCESS_KEY`
- `MINIO_SECRET_KEY`
- `JWT_SECRET`
- `JWT_EXPIRATION` (seconds, default `3600`)
- `OTLP_ENDPOINT`
- `LOG_LEVEL` (default `info`)
- `ENVIRONMENT` (default `development`)

### Example `config.yaml`
```yaml
HTTP_ADDR: ":8080"
GRPC_ADDR: ":9090"
DATABASE_URL: "postgres://user:pass@localhost:5432/app?sslmode=disable"
REDIS_URL: "redis://localhost:6379"
KAFKA_BROKERS:
  - "localhost:9092"
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

## Notes
- `KAFKA_BROKERS` accepts a YAML list or comma-separated env value.
- Env vars always override YAML.
- Keep secrets out of git; prefer env vars or secret managers.

## License
Proprietary. Internal use only.
