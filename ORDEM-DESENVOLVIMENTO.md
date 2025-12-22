## Ordem sugerida (incremental e testável)

-  **Tenant/identidade primeiro:** consolidar autenticação/multi-tenant para liberar RLS e escopar eventos. Priorize auth-gateway e tenant-manager (em paralelo, mas finalize tenant-manager antes de depender dele).
-  **Plano/receita cedo:** billing-service logo após auth/tenant para garantir fluxo de onboarding e limites de uso.
-  **Dados e eventos-base:** platform-events (lib) já está; conecte analytics-query-service apenas depois de ter ingestão mínima funcionando.
-  **Execução de agentes e ferramentas:** agent-orchestrator → tools-gateway (depende de auth/tenant) → voice-gateway (telephony é mais caro para iterar; só depois de fluxos principais).
-  **Frontends:** console/mfes depois que auth/tenant/billing expõem endpoints estáveis; marketing site independente pode seguir em paralelo.

## Quando trazer Python

- **Analytics Processor (Python):** só após haver eventos Kafka consistentes com tenant_id (saída dos gateways/orchestrator). Ideal: depois que agent-orchestrator + tools-gateway já publicam eventos mínimos.
- **Reporting Export (Python):** após ClickHouse ter esquema e dados ingeridos (via processor) e queries estáveis; portanto, depois do processor.
- **Libs Python (analytics-common, nlp-utils):** podem ser preparadas em paralelo ao processor, mas evite integrar em prod antes dos schemas/contratos de evento estarem fixos.

## Sequência prática (fases curtas)

1. **Fundação de acesso:** auth-gateway + tenant-manager + aplicar middleware de auth/tenant em libs/serviços.
2. **Monetização básica:** billing-service (webhooks Stripe + quotas iniciais).
3. **Orquestração de agentes:** agent-orchestrator com eventos Kafka mínimos (incluindo tenant_id).
4. **Ferramentas externas:** tools-gateway consumindo auth/tenant e emitindo eventos.
5. **Voz:** voice-gateway depois dos fluxos principais (depende de orchestrator/tools).
6. **Ingestão/analytics (Python):** analytics-processor-service assim que eventos estiverem prontos; definir esquema ClickHouse e tópicos Kafka.
7. **Consultas:** analytics-query-service lendo ClickHouse após processor.
8. **Export/relatórios (Python):** reporting-export-service quando ClickHouse já estiver populado.
9. **Frontends:** console/MFEs para expor auth/billing e visualizar métricas; marketing site separado.

## Critérios de prontidão antes de cada etapa

1. Sempre: contratos de evento/documentação + testes automatizados (unitários) e lint.
2. Antes do processor: tópicos Kafka, chaves por tenant_id, esquema de evento versionado.
3. Antes do query-service: tabelas ClickHouse criadas e populadas por um ambiente de teste.
4. Antes do export: queries finalizadas e SLAs de volume.