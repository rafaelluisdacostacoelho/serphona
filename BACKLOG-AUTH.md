# Backlog de Adoção do platform-auth (pt-BR)

Propósito: plano coordenado para adotar o platform-auth nos serviços com RLS/tenant, envelope e autorização consistentes.

## Princípios norteadores
- Multi-tenant primeiro: reforçar tenant_id em contexto, headers (EnsureTenantHeader), queries de DB e eventos Kafka; falhar fechado se ausente.
- Envelope e erros uniformes em Gin/chi/net/http/gRPC usando os helpers de response; trace/request IDs sempre presentes.
- Autorização por escopo: RequiredScopes por rota e tokens service-to-service com audience separada; propagar cabeçalhos de identidade de serviço no outbound.
- Observabilidade e segurança: métricas/tracing sem vazar segredos; redação de headers; parsing estrito e limites de tamanho; pronto para TLS/mTLS.
- Testes como guarda: golden tokens, contrato de envelope, propagação de tenant, retries/TLS e propagação de headers outbound.

## Ordem de execução (fases)

### Fase 0 — Base da lib
- [x] Publicar orientação e rollout de tokens service-to-service (AUTH-GUIDANCE en/pt-BR).
- [x] Helpers de tenant para DB/Kafka e transportes outbound entregues no platform-auth.
- [x] Seção de exemplos dos helpers de envelope em Gin/chi/gRPC finalizada e referenciada nos serviços.
- [x] Helper de autorização de cliente para chamadas internas adotado onde necessário (identidade do serviço + scopes).

### Fase 1 — Testes de contrato do envelope
- [x] Auth-gateway e tenant-manager: handlers HTTP usam helpers de envelope; adicionar testes de contrato para sucesso/erro + metadata de trace_id.
- [x] Superfícies gRPC (tenant-manager) mapeiam status/metadata via interceptors do platform-auth; testes de contrato prontos. Agent-orchestrator não expõe gRPC no momento (N/A).

### Fase 2 — Propagação de tenant (ordem)
- [x] tenant-manager: helpers de tenant nos repositórios; EnsureTenantHeader no outbound/Kafka; testes table-driven de injeção de header. (progresso: repos + testes de validação em repos/handlers ✅; header Kafka coberto por ensureTenantHeaders + testes)
- [x] billing-service: guard rails de tenant em DB/Kafka; HTTP outbound usa transporte do platform-auth + EnsureTenantHeader; adicionar testes. (DB já usa EnforceTenant/TenantIDFromContext; Kafka producer/DLQ agora injeta X-Tenant-Id com testes; não há HTTP outbound hoje)
- [x] analytics-query-service: helpers de tenant no caminho de consulta; outbound (quando existir) valida header de tenant; adicionar testes. (tenantIDFromContext já enforça via GetTenantIDFromContext e EnsureTenantHeader; sem HTTP outbound; testes de propagação adicionados)
- [x] tools-gateway: todos os clients outbound usam transporte do platform-auth + EnsureTenantHeader; guard rails de tenant nos fluxos de execução; adicionar testes. (SOAP/GraphQL/HTTP clients agora injetam X-Tenant-Id via middleware.TenantIDFromContext; handler ExecuteTool propaga tenant context via WithTenantID; testes table-driven para injeção de header HTTP ✅)
- [x] auth-gateway: auditar chamadas outbound (se houver) para garantir transporte do platform-auth + EnsureTenantHeader; adicionar testes. (Apple OAuth não requer tenant; CreateTenant é stub sem implementação; verificado ✅)
- [x] voice-gateway: propagar header de tenant nos clients de agente/tenant e eventos Kafka; testes ao iniciar conversa. (Agent/tenant clients já usam authclient.WithDefaultTransport; callService propaga WithTenantID antes de chamar agentes; testes de agent client adicionados ✅)
- [x] rag-gateway: verificar que a injeção de header de tenant continua correta; adicionar testes se faltar. (Embedding client para OpenAI; authclient.WithDefaultTransport em uso; X-Tenant-Id para chamadas internas incluído; verificado ✅)
- [ ] Cross-service: testes de integração assegurando X-Tenant-Id em chamadas outbound quando o tenant está no contexto.

### Fase 3 — Auth/resiliência no outbound
- [ ] Garantir que todos os serviços definem env de identidade do serviço e, quando necessário, usam o helper de auth de cliente em chamadas internas (tokens com scopes, SERVICE_AUDIENCE).
- [ ] Verificar retries/backoff + circuit breaker via client do platform-auth onde usado; cobrir mapeamento de erros.

### Fase 4 — Observabilidade e segurança
- [ ] Redação e limpeza de logs nos serviços usando helpers do platform-auth; verificar se nenhum header sensível vaza.
- [ ] Métricas Prometheus de auth (sucesso/latência) expostas em todos os transports; dashboards/alertas para falhas de auth e tenant ausente.
- [ ] Validação de orientação CSRF para fluxos com cookie (onde aplicável); defaults fail-closed conferidos nos serviços.

### Fase 5 — Rollout e docs
- [ ] Referenciar este backlog nos backlogs específicos dos serviços; manter status sincronizado.
- [ ] Atualizar exemplos em docs/architecture por serviço após a adoção (en/pt-BR).
- [ ] Adicionar notas de release cobrindo formato de claims/comportos de middleware e eventuais breaking changes.
