# Biblioteca Platform Core

Utilitários centrais compartilhados pelos serviços da Serphona. Atualmente oferece carregamento de configurações com defaults usando Viper.

## Funcionalidades
- Leitura centralizada de config via YAML + variáveis de ambiente.
- Defaults para ajustes comuns (portas HTTP/GRPC, nível de log, expiração do JWT, porta do ClickHouse).
- Struct `Config` tipada para os serviços consumirem.
- Helper de validação para garantir chaves obrigatórias.
- Helper de logger (zap) que respeita o nível configurado e pode incluir metadados do serviço.
- Helper de health (liveness/readiness).
- Helper de segredos com provider configurável (env por padrão).

## Instalação
```bash
go get github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-core
```

## Uso rápido
```go
package main

import (
    "log"

    "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-core/config"
)

func main() {
cfg, err := config.Load()
if err != nil {
    log.Fatalf("config: %v", err)
}

log.Printf("HTTP em %s, env: %s, log: %s", cfg.HTTPAddr, cfg.Environment, cfg.LogLevel)

if err := config.ValidateRequired(cfg, "DATABASE_URL", "JWT_SECRET"); err != nil {
    log.Fatalf("faltam configs obrigatorias: %v", err)
}

logger, err := logger.NewWithMeta(cfg.LogLevel, "billing-service", cfg.Environment, "1.2.3")
if err != nil {
    log.Fatalf("logger: %v", err)
}
defer logger.Sync()

logger.Info("servico iniciando", zap.String("env", cfg.Environment))
}
```

## Health checks
```go
mux := http.NewServeMux()
mux.HandleFunc("/health", health.Handler(
    func() error { return nil }, // liveness
    func() error { return nil }, // readiness (db, cache, etc.)
))
```

## Segredos
```go
dbPass, err := secrets.Get("DB_PASSWORD")
if err != nil {
    log.Fatal(err)
}

// Plugar provider customizado (ex.: secret manager):
secrets.SetProvider(meuProvider)
```

## Configuração
Ordem de leitura:
1. Defaults (abaixo)
2. `config.yaml` (cwd ou `/etc/serphona/`)
3. Variáveis de ambiente (sobrepõem)

### Chaves suportadas
- `HTTP_ADDR` (default `:8080`)
- `GRPC_ADDR` (default `:9090`)
- `DATABASE_URL`
- `REDIS_URL`
- `KAFKA_BROKERS` (env separado por vírgula funciona)
- `CLICKHOUSE_HOST`
- `CLICKHOUSE_PORT` (default `8123`)
- `MINIO_ENDPOINT`
- `MINIO_ACCESS_KEY`
- `MINIO_SECRET_KEY`
- `JWT_SECRET`
- `JWT_EXPIRATION` (segundos, default `3600`)
- `OTLP_ENDPOINT`
- `LOG_LEVEL` (default `info`)
- `ENVIRONMENT` (default `development`)

### Exemplo `config.yaml`
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
JWT_SECRET: "mude-este-valor"
JWT_EXPIRATION: 3600
OTLP_ENDPOINT: "http://localhost:4317"
LOG_LEVEL: "info"
ENVIRONMENT: "development"
```

## Observações
- `KAFKA_BROKERS` aceita lista YAML ou env separado por vírgula.
- Variáveis de ambiente sempre sobrepõem YAML.
- Mantenha segredos fora do git; prefira env vars ou secret managers.

## Licença
Proprietária. Uso interno apenas.
