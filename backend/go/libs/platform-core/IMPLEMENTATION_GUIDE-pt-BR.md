# Platform Core - Guia de Implementacao

Guia para adotar a biblioteca platform-core nos seus servicos.

## O que voce recebe
- Struct tipada `Config` cobrindo HTTP/GRPC, database, Redis, Kafka, ClickHouse, MinIO, JWT, OTLP, nivel de log, ambiente.
- Loader que mescla defaults, `config.yaml` e variaveis de ambiente (suporta brokers Kafka separados por virgula via env).
- Helper de validacao para chaves obrigatorias.
- Helper de logger (zap) que respeita `LOG_LEVEL`.
- Handler de health para liveness/readiness.
- Helper de segredos via env.

## Passos de setup
1) Adicione a dependencia no `go.mod` do servico:
```go
require github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-core v1.0.0
```
Rode `go mod tidy`.

2) Crie `config.yaml` (opcional) na raiz do servico ou em `/etc/serphona/`.

3) Defina variaveis de ambiente para segredos/overrides (lista abaixo).

4) Carregue a config na inicializacao:
```go
cfg, err := config.Load()
if err != nil {
    log.Fatalf("config: %v", err)
}

if err := config.ValidateRequired(cfg, "DATABASE_URL", "JWT_SECRET"); err != nil {
    log.Fatalf("config faltando: %v", err)
}

logger, err := logger.New(cfg.LogLevel)
if err != nil {
    log.Fatalf("logger: %v", err)
}
defer logger.Sync()

// Endpoint de health
mux := http.NewServeMux()
mux.HandleFunc("/health", health.Handler(
    func() error { return nil }, // liveness
    func() error { return nil }, // readiness checks
))
```

5) Use `cfg` na aplicacao (portas, URLs, brokers, etc.).

## Variaveis de ambiente (defaults)
- `HTTP_ADDR` (`:8080`)
- `GRPC_ADDR` (`:9090`)
- `DATABASE_URL`
- `REDIS_URL`
- `KAFKA_BROKERS` (env separado por virgula funciona)
- `CLICKHOUSE_HOST`
- `CLICKHOUSE_PORT` (`8123`)
- `MINIO_ENDPOINT`
- `MINIO_ACCESS_KEY`
- `MINIO_SECRET_KEY`
- `JWT_SECRET`
- `JWT_EXPIRATION` segundos (`3600`)
- `OTLP_ENDPOINT`
- `LOG_LEVEL` (`info`)
- `ENVIRONMENT` (`development`)

## Exemplo de `config.yaml`
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
JWT_SECRET: "mude-este-valor"
JWT_EXPIRATION: 3600
OTLP_ENDPOINT: "http://localhost:4317"
LOG_LEVEL: "info"
ENVIRONMENT: "development"
```

## Checklist de implementacao
- [ ] Adicionar dependencia e rodar `go mod tidy`
- [ ] Criar `config.yaml` (opcional) sem segredos
- [ ] Definir env vars para segredos (`DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, etc.)
- [ ] Carregar config no start com `config.Load()`
- [ ] Validar chaves obrigatorias com `config.ValidateRequired(...)`
- [ ] Inicializar logger zap com `logger.New(cfg.LogLevel)`
- [ ] Expor `/health` usando `health.Handler(...)`
- [ ] Usar `secrets.Get` para segredos em env (ou secret manager) quando necessario
- [ ] Ligar portas e clients usando os valores carregados
- [ ] Documentar quais chaves o servico exige
- [ ] Adicionar testes para validar env/fields obrigatorios (opcional)

## Troubleshooting
- Sem arquivo: o loader funciona sem `config.yaml`; use env/defaults.
- Sem env: defina a variavel ou coloque no arquivo de config.
- Kafka via env: `KAFKA_BROKERS=host1:9092,host2:9092`.

## Suporte
Uso interno — fale com o time de plataforma se precisar de novas chaves ou validacao adicional.
