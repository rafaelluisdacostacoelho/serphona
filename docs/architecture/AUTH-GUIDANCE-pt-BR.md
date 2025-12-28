# Orientações de Autenticação — Tokens de refresh/serviço, CSRF e fail-closed (pt-BR)

## Objetivos
- Definir padrões para tokens de refresh/serviço e comportamentos CSRF/fail-closed em todos os serviços Serphona.
- Manter isolamento multi-tenant (tenant_id em tudo) e aderir aos contratos do platform-auth.

## Tokens serviço-a-serviço (client credentials)
- Audience: emitir com `SERVICE_AUDIENCE` separada da `AUDIENCE` de usuário.
- Claims: incluir `tenantId` (ou tenant "platform" para infra), `service` (ID do chamador), `scopes` mínimos por ação interna; `exp` curta (5–15m), `nbf` recomendado.
- Issuer: auth-gateway/issuer central; `iss` consistente.
- Chaves: preferir JWKS (`JWKS_URL`, `kid` permitido); rotacionar; HMAC só como fallback dev.
- Validação: `AllowedAlgs` RS256/ES256; `Audience` = `SERVICE_AUDIENCE` por serviço.
- Propagação: sempre encaminhar `X-Request-Id`, `Traceparent` e `X-Tenant-Id`; use o transporte HTTP instrumentado (já propaga request/trace/tenant).
- Mínimo privilégio: scopes internos específicos (ex.: `tenant-manager:read-tenants`, `billing:write-invoices`); rejeitar tokens internos sem scope.

## Tokens de refresh (usuário)
- Armazenamento: cookies httpOnly, Secure; SameSite=Lax por padrão; rotacionar a cada refresh (novo refresh+access).
- Audience: `AUDIENCE` de usuário; não reutilizar para serviço-a-serviço.
- Tempo de vida: refresh 7–30d; access 5–15m; invalidar em logout/rotação.
- Binding: atrelar refresh a `sessionId`, `tenantId` e opcionalmente hash de IP/UA; revogar em divergência.
- Revogação: manter store (Redis/DB) por família; revogar em logout ou suspeita de comprometimento.

## CSRF e fail-closed
- CSRF para cookies: double-submit (`X-CSRF-Token` igual ao cookie) ou SameSite=Strict; falta/erro => 403.
- Métodos seguros: permitir GET/HEAD/OPTIONS; exigir CSRF em POST/PUT/PATCH/DELETE.
- Fail-closed: se secret/JWKS/config faltam, retorne 500/401 (nunca 200); middleware não deve panicar.
- Redação de cabeçalhos: manter Authorization/Cookie/Proxy-Authorization redigidos em logs e campos seguros.

## Checklist operacional dos serviços
- Configurar: `ISSUER`, `AUDIENCE`, `SERVICE_AUDIENCE`, `JWKS_URL`, `JWKS_ALLOWED_KIDS`, `JWT_SECRET` (fallback), `REQUIRED_SCOPES` por rota.
- Saída: usar transporte HTTP do platform-auth; definir cabeçalhos de nome/instância do serviço; confiar na propagação automática de request/trace/tenant.
- Entrada: aplicar tenant via middleware/helpers; alinhar envelopes de resposta; expor /healthz + métricas.
- Rotação: monitorar erros de fetch JWKS; definir `JWKS_CACHE_TTL` adequado.

## Plano de rollout (serviço-a-serviço)
1) Definir matriz de scopes por serviço (produtor/consumidor).
2) Adicionar `SERVICE_AUDIENCE` na config dos serviços; ajustar ValidationConfig.Audience.
3) Habilitar JWKS com kids permitidos; rotacionar chaves; remover HMAC interno após JWKS estável.
4) Instrumentar chamadas outbound com cabeçalhos propagados; testar forwarding de tenant/request ID.

## Plano de rollout (refresh/CSRF)
1) Aplicar cookies Secure+httpOnly; SameSite=Lax (Strict se aceitável).
2) Implementar token CSRF (double-submit) em rotas que mudam estado.
3) Rotacionar refresh a cada uso; armazenar família para revogação; adicionar logout que revoga.
4) Testes: falta/erro de CSRF -> 403; falta de secret/JWKS -> 500/401 fail-closed.
