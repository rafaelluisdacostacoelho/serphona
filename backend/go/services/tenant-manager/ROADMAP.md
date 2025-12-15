## Tenant Manager - Próximos Passos (estado atual)

### Concluído
- Entrypoint e bootstrap (`cmd/server`), gRPC/HTTP com interceptors e JWT (HS/RS).
- Domínio + app de tenants e API keys, repositórios Postgres, cache Redis tolerante a nil e publisher Kafka opcional.
- Migrações 000001 (tenants) + 000002 (api_keys) com runner idempotente e down.
- Config alinhada (`ENVIRONMENT`, `DATABASE_*`, `KAFKA_*`, JWT, métricas/tracing) e `.env.example`/READMEs atualizados.
- Health/ready/metrics expostos; keepalive gRPC; testes básicos (API keys domain, pagination).
- OpenAPI placeholder (`api/openapi/openapi.yaml`) e script de proto corrigido.

### Em aberto / a fazer
- DTOs/validators HTTP dedicados (`internal/adapter/http/dto` e `validator`) ou mover handlers para Gin puro.
- Migration 000003 para configs adicionais de tenant (up/down).
- Scripts utilitários em `scripts/` (migrate.sh, generate.sh, seed.sh) conforme estrutura documentada.
- Swagger/UI ou geração automática da OpenAPI (atual hoje é estática).
- Publisher/cache no-op explícitos para rodar sem Redis/Kafka sem logs de erro.
- Testes adicionais (handlers HTTP/gRPC e repositórios) para cobrir fluxos principais.
