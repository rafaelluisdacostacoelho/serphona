# Backlog Global — Plataforma de Voz (pt-BR)

Propósito: sequenciar o trabalho entre serviços/libs para que o loop de voz (STT → agente → TTS) fique robusto, multi-tenant e observável.

## Etapas (seguir na ordem)
1) Voice Gateway — loop central e segurança  
   - Backlog: [backend/go/services/voice-gateway/backlog.md](../../backend/go/services/voice-gateway/backlog.md)  
   - Escopo: selecionar provider STT/TTS por tenant, abrir conversa via agent-orchestrator, fazer streaming de áudio (PCM 16k mono) no fluxo STT→agente→TTS e tocar de volta. Adicionar auth/idempotência nos webhooks, propagação de tenant e envelopes de erro.  
   - Saída: testes automatizados com fakes (Asterisk/Redis/Kafka/agent/tenant), métricas/traces, readiness falha quando dependências caem.

2) Tenant Manager — config de provider e DID lookup  
   - Backlog: [backend/go/services/tenant-manager/backlog.md](../../backend/go/services/tenant-manager/backlog.md)  
   - Escopo: resolver DID→tenant com auth e envelope; expor config de provider STT/TTS/LLM por tenant com `X-Tenant-Id`; testes de contrato; rate limiting e mapeamento de erros.  
   - Saída: API estável consumida pelo voice-gateway; testes travam contrato.

3) Agent Orchestrator — tokens de serviço e SLAs  
   - Backlog: [backend/go/services/agent-orchestrator/backlog.md](../../backend/go/services/agent-orchestrator/backlog.md)  
   - Escopo: exigir tokens SERVICE_AUDIENCE, timeouts/retries, envelope com trace/request ID; garantir idempotência de create/turn/end e retorno de metadados do agente (nome/voz).  
   - Saída: stub de integração no voice-gateway passa sem flaps de retry; defaults seguros para carga.

4) Platform Auth — adoção do transporte  
   - Backlog: [backend/go/libs/platform-auth/BACKLOG.md](../../backend/go/libs/platform-auth/BACKLOG.md)  
   - Escopo: garantir que clientes outbound em voice-gateway/tenant-manager/agent-orchestrator usem o transporte da lib com propagação de tenant/request/trace e tokens SERVICE_AUDIENCE.  
   - Saída: headers outbound verificados em testes.

5) Observabilidade e Infra  
   - Backlogs: [backend/go/libs/platform-observability](../../backend/go/libs/platform-observability), [infra/helm](../../infra/helm), [infra/terraform](../../infra/terraform)  
   - Escopo: métricas para ARI/STT/TTS/chamadas ao agente, contadores de estado de chamada, readiness (Redis/Kafka/Asterisk), valores de Helm para segredos/creds, HPA e tracing exporter.  
   - Saída: readiness falha de forma fechada; dashboards/alertas cobrem latência/orçamento de erro.

## Como usar
- Entregar as tarefas do backlog do projeto correspondente a cada etapa; só avançar para a próxima quando os critérios de saída forem cumpridos. Espelhar qualquer ajuste também nesta versão PT-BR.
