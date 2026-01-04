# Notas de Release: Platform-Auth

## Resumo
Esta release introduz melhorias significativas na autenticação e autorização multi-tenant, incluindo suporte a métricas cross-service, proteção CSRF e integração com o platform-auth.

## Novidades
- **Métricas Cross-Service**: Adicionadas métricas padronizadas para autenticação, incluindo `auth_requests_total` e `auth_request_duration_seconds_bucket`.
- **Proteção CSRF**: Middleware implementado para proteger fluxos baseados em cookies.
- **Integração com Platform-Auth**: Serviços agora utilizam `EnsureTenantHeader` e validação de escopos para autenticação consistente.

## Breaking Changes
- **Formato de Claims**: Tokens JWT agora incluem `tenant_id` e `service` como claims obrigatórios.
- **Middleware**: Serviços devem adotar o middleware `Authenticate` e `CSRFProtectionMiddleware`.

## Recomendações
- Atualize os serviços para utilizar as novas métricas e middlewares.
- Verifique os dashboards no Grafana para monitorar as novas métricas.
- Consulte os exemplos atualizados em `docs/architecture` para integração.

## Documentação
- [Exemplos de Métricas e Alertas](../architecture/AUTH-METRICS-OBSERVABILITY-pt-BR.md)
- [Backlog de Adoção](../../BACKLOG-AUTH.md)