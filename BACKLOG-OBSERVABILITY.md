# Backlog de Observabilidade

Pendências para alinhar métricas/alertas com a instrumentação real dos serviços.

- Tenant Manager: ajustar o ServiceMonitor para coletar na porta/endpoint real de métricas (hoje 9091 `/metrics`), senão os alertas ficam sem dados.
- Agent Orchestrator: instrumentar HTTP/auth no Prometheus (contador de requisições com status/outcome, histograma de duração, labels de tenant/serviço) e depois reativar alertas de erro/latência.
- Billing Service: instrumentar HTTP/auth no Prometheus (contador de requisições com status/outcome, histograma de duração, labels de tenant/serviço) e depois reativar alertas de erro/latência.
- Tools Gateway: adicionar métricas específicas de auth (sucesso/falha/tenant ausente) e alertas; manter métricas HTTP já existentes.
- Cross-service: padronizar o esquema de métricas de auth (sucesso/falha, histograma de latência, `tenant_id`, `service`, `component`) e configurar dashboards/alertas para falhas de auth e tenant ausente.
