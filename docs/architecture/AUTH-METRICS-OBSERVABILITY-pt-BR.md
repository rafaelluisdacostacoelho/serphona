# Métricas e Alertas do Auth Gateway (auth-gateway)

Status: Rascunho
Owner: Plataforma/Auth
Atualizado em: 2026-01-04

## Escopo
Instrumentação, dashboard e alertas para fluxos de autenticação no auth-gateway, além de guardrails cross-service para falhas de auth e ausência de tenant usando as métricas compartilhadas do platform-auth.

## Métricas
- `auth_gateway_auth_events_total{event,outcome}`: contador de eventos de auth por desfecho.
- `auth_gateway_auth_duration_seconds_bucket{event,outcome,le}`: histograma de latência dos fluxos de auth.
- `auth_gateway_tenant_missing_total{path}`: contador de requisições sem `tenant_id` em contexto.
- Cross-service (platform-auth) counters/histogramas: `auth_requests_total{transport,result,service,tenant_id,component}` e `auth_request_duration_seconds_bucket{transport,result,service,component,le}` para pivôs por serviço/componente/tenant.

## Dashboard
Arquivo: `backend/go/services/auth-gateway/grafana/auth-gateway-auth.json`
Painéis:
- Taxa de falhas (5m) em stat
- Eventos por desfecho (rate)
- Eventos por evento/desfecho (rate)
- Latência p95/p99 por evento (sucesso)
- Tenant ausente por path (rate 5m)
- Razão de falhas por evento

Variável de datasource: `DS_PROMETHEUS`.

Grafana cross-service (novo JSON a colocar em `grafana/cross-service-auth.json`):
- Taxa de falha por serviço/componente com `sum by (service,component,result)(rate(auth_requests_total{result!="ok"}[5m]))`
- Taxa de tenant ausente por serviço/componente com `sum by (service,component)(rate(auth_requests_total{result="unauthorized",tenant_id="unknown"}[5m]))`
- Split sucesso vs falha por tenant com `sum by (tenant_id,service,result)(rate(auth_requests_total[5m]))`
- Latência p95 por serviço/componente usando `histogram_quantile(0.95, sum by (le,service,component)(rate(auth_request_duration_seconds_bucket[5m])))`
- Top tenants por falhas de auth via `topk(5, sum by (tenant_id,service)(rate(auth_requests_total{result!="ok"}[5m])))`

## Alertas (PrometheusRule)
Arquivo: `infra/helm/auth-gateway/templates/prometheusrule.yaml` (habilitado via values).
- AuthGatewayHighFailureRate: razão de falha > 10% por 10m.
- AuthGatewayLatencyHighP95: p95 > 1s por 10m.
- AuthGatewayLatencyHighP99: p99 > 2s por 10m.
- AuthGatewayTenantMissing: qualquer ocorrência de tenant ausente em 5m.
Limiares configuráveis nos values do Helm.

PrometheusRule cross-service (adicionar em `infra/helm/platform-observability/templates/prometheusrule-auth.yaml`):
- AuthFailuresByServiceHigh: `sum by (service,component)(rate(auth_requests_total{result!="ok"}[5m])) > 1` por 10m; rótulos: service/component/tenant quando disponíveis.
- AuthTenantMissingHigh: `sum by (service,component)(rate(auth_requests_total{tenant_id="unknown"}[5m])) > 0` por 5m para detectar falta de propagação de tenant.
- AuthLatencyP95High: `histogram_quantile(0.95, sum by (le,service,component)(rate(auth_request_duration_seconds_bucket[5m]))) > 1` por 10m.

## Toggles no Helm
`infra/helm/auth-gateway/values*.yaml`:
- `serviceMonitor.enabled`: expõe métricas para scrape pelo Prometheus Operator.
- `prometheusRule.enabled`: instala as regras de alerta.
- Limiares e severidades em `prometheusRule.*`.

## Notas de adoção
- Métricas são emitidas pelos handlers HTTP e middleware do auth-gateway e pelo middleware platform-auth (Gin/HTTP/gRPC) nos demais serviços. O rótulo de serviço é injetado no startup via `SetAuthMetricsService` e os rótulos de tenant fluem do token/header.
- Garantir Prometheus Operator e scrape nos namespaces de todos os serviços que usam platform-auth.
- Importar o JSON no Grafana apontando para a datasource Prometheus e adicionar o dashboard cross-service quando disponível.

## Próximos passos
- Replicar o padrão para outros serviços com métricas/SLOs específicos (tenant-manager, billing, tools, voice, rag).
- Adicionar recording rules para SLIs/SLAs quando necessário (ex.: disponibilidade/erro budget de login).
- Integrar alertas às rotas do Alertmanager por ambiente (stg/prod) e on-call.

## Exemplos Atualizados

### Exemplo de Métrica Cross-Service
```yaml
- name: auth_requests_total
  help: Contador de requisições de autenticação por serviço e tenant.
  labels:
    - transport
    - result
    - service
    - tenant_id
    - component
```

### Exemplo de Configuração PrometheusRule
```yaml
- alert: AuthGatewayHighFailureRate
  expr: |
    sum(rate(auth_requests_total{service="auth-gateway", result!="ok"}[5m]))
      / sum(rate(auth_requests_total{service="auth-gateway"}[5m])) > 0.1
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Taxa de falhas alta no Auth Gateway"
    description: "A taxa de falhas ultrapassou 10% nos últimos 10 minutos."
```
