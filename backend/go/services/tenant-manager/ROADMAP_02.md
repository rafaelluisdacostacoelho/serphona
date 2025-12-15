Aqui está o que encontrei de lacunas/bloqueios para fechar o tenant-manager:

## Bloqueadores

[x] - Rotas HTTP quebradas: cmd/server/main.go sobe Gin, mas os handlers são escritos para chi (internal/adapter/http/handler/tenant.go usa chi.URLParam e request_id no Context). Resultado: path params nunca são lidos, o request_id não chega, e as rotas de tenants não funcionam. O router chi em internal/adapter/http/router/router.go nem é usado.
[x] - Cache/eventos sem fallback: internal/application/tenant/service.go chama s.cache.* e s.eventPublisher.* sem checar nil. Se Redis ou Kafka falharem (ou estiverem desabilitados, como no próprio main.go), o serviço panica nas operações básicas.
[x] - API Keys não integradas: o service é criado com apiKeyRepo nil em cmd/server/main.go, rotas de API key estão comentadas, e internal/adapter/postgres/apikey_repo.go não implementa a interface de domínio (internal/domain/apikey/repository.go). Ou seja, a feature está incompleta/invocável.
    Integração inicial de API Keys feita e build passando:
    - cmd/server/main.go: instanciei APIKeyRepo e injetei no tenant.NewService; reativei as rotas de API keys com wrappers Gin.
    - internal/adapter/postgres/apikey_repo.go: agora gera chaves com hash SHA‑256 e prefixo compatíveis com o schema de api_keys, grava hash/prefixo e valida por hash (verifica expiração/revogação).
    - internal/adapter/http/handler/apikey.go: acrescentei handler de DELETE (ainda placeholder).
    - go test ./...: compilou sem falhas (sem testes definidos).
    A feature ainda é básica (handlers retornam placeholders), mas o serviço já não fica com repo nil e validações de API key usam o armazenamento real. Próximo passo natural: expor um serviço de API keys real (usar internal/application/apikey.Service + repository completo)
[x] - Migrações: só existe migrations/000001_create_tenants.up.sql; não há down nem migrações sequenciais. E internal/adapter/postgres/migrations.go é apenas um placeholder, então AutoMigrate não executa nada.
[x] - Segurança placeholder: internal/adapter/http/middleware/gin_middleware.go aceita qualquer token >10 chars. Sem validação JWT real, todas as rotas autenticadas ficam expostas.

## Importantes

[x] - Health/ready/metrics: o README promete /health, /ready, /metrics, mas só há handlers chi (internal/adapter/http/handler/health.go) e o main usa handlers inline no Gin; não há /metrics.
[x] - Config vs .env: internal/config/config.go espera ENVIRONMENT, SERVER_HOST, etc., mas .env.example usa ENV, SERVER_PORT 8082, chaves DB_*, CORS_* que não são lidas. Vai carregar defaults errados sem mapear essas variáveis.
[x] - Go toolchain: go.mod declara go 1.24.0, mas o Dockerfile usa golang:1.21-alpine. Build vai divergir ou falhar conforme a versão instalada.
[x] - gRPC parcial: só há handler (internal/adapter/grpc/handler/tenant.go), sem interceptors de auth/rls, sem keepalive/health refinados; GRPCKeepaliveConfig do config não é aplicado no startGRPCServer.
[x] - Paginação/utilitários: não existe pkg/pagination ou middleware de correlação como descrito na estrutura da doc; todas as middlewares de correlação/log chi não são usadas pelo Gin.

## Desejáveis

[x] - Docs/geração: não há api/openapi/openapi.yaml; Makefile generate-openapi aponta para um caminho inexistente. Proto está ok, mas o script scripts/generate-proto.sh usa um echo estranho e cria pasta proto/tenantpb que não é usada pelos arquivos atuais.
[x] - Observabilidade: config expõe MetricsConfig, mas não há Prometheus handler; tracing/feature flags de .env.example não aparecem no código.
[x] - Tests: nenhum _test.go; não há cobertura básica dos casos de domínio/handlers.

Próximos passos recomendados: (1) decidir se o HTTP fica em Gin ou chi e alinhar handlers/middlewares/routers; (2) adicionar no-op cache/publisher para operar sem Redis/Kafka e ligar de fato API keys (repo compatível + wiring no main + rotas); (3) arrumar migrações/down e execução real; (4) corrigir .env.example/config e versão Go; (5) adicionar endpoints de saúde/metrics coerentes e testes mínimos de domínio/handlers.