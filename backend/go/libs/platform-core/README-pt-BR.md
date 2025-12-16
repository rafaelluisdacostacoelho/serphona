# Biblioteca Platform Core

Utilitários centrais compartilhados pelos serviços da Serphona. Atualmente oferece carregamento de configurações com defaults usando Viper.

## Funcionalidades
- Leitura centralizada de config via YAML + variáveis de ambiente.
- Defaults para ajustes comuns (portas HTTP/GRPC, nível de log, expiração do JWT, porta do ClickHouse).
- Struct `Config` tipada para os serviços consumirem.

## Instalação
```bash
go get github.com/serphona/serphona/backend/go/libs/platform-core
```

## Uso rápido
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

    log.Printf("HTTP em %s, env: %s, log: %s", cfg.HTTPAddr, cfg.Environment, cfg.LogLevel)
}
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
