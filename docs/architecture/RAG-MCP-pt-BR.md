# Arquitetura RAG + MCP para o Serphona (pt-BR)

## Por que RAG + MCP
- RAG entrega conhecimento atualizado e por tenant (playbooks, políticas, catálogos), reduzindo alucinação e forçando citações.
- MCP padroniza ações com schemas tipados, authz e trilha de auditoria, evitando tooling frágil guiado só por prompt.
- Voz exige baixa latência, previsibilidade e explicabilidade (o que foi dito, por quê, com base em quê, que ações rodaram).

## Onde se encaixam
- **Camada de Voz Realtime**: SIP ingress (Kamailio) → rtpengine → Asterisk/WebRTC → STT/TTS → Turn Manager.
- **Runtime do Agente**: Orquestrador/router, estado de sessão, guardrails/políticas.
- **Camada de Conhecimento (RAG)**: Ingestão → chunking → embeddings → vector store com filtros por tenant → re-rank → geração com citações; cache quente de KB.
- **Camada de Ação (MCP)**: MCP servers por domínio expondo tools tipadas; authz/políticas por tenant/agente/ambiente; logging + idempotência.
- **Dados & Observabilidade**: Kafka para eventos; ClickHouse para analytics; S3/MinIO para artefatos (transcrições, logs de tool, snapshots de retrieval); platform-observability para traces/métricas/logs.

## Arquitetura de RAG
- **Ingestão/Governança**: Trate KB como produto. Normalizar, fragmentar, embedar, indexar; metadados: tenant_id, namespace/domínio, idioma, canal, valid_from/valid_to, versão/etag, acl/tags.
- **Storage**: Vector DB com filtros rígidos por tenant (ou índice/namespace por tenant); blobs em S3/MinIO; eventos de ingestão no Kafka.
- **Retrieval**: Filtros tenant_id + canal + domínio + idioma + validade; topK baixo, re-ranker, threshold de similaridade; citações com IDs de trechos; cache quente com invalidação por document_id.
- **Separação**: Memória da call (estado curto) em store transacional (Redis/DB) vs KB corporativa no vector store.

## Arquitetura de MCP
- **Padrão**: MCP por domínio (recomendado) — mcp-crm, mcp-ticketing, mcp-billing, mcp-telephony-control, mcp-analytics.
- **Contratos**: Entradas/saídas tipadas, códigos de erro, idempotência onde fizer sentido; exemplos: crm.lookup_customer, ticket.create, billing.get_open_invoices, telephony.transfer_call, messaging.send_whatsapp, analytics.log_event, rag_query.
- **Políticas**: Engine de regras por tenant_id, agent_role, environment (dev/hml/prd), allow/deny por tool; rate limit e quotas por tenant.
- **Auditoria/Obs**: Logar chamadas (tenant_id, agent_id, call_id, tool, hash de input/output, duração, status); tracinhos em todas as calls; alinhar com billing.

## Fluxo de turn (voz)
Áudio → STT → Router (intenção/risco/idioma) → RAG (com filtros) → LLM com citações → MCP (se precisar agir) → validação/log/state → TTS. Exemplo “segunda via do boleto”: router marca billing → RAG puxa política + passos ERP → plano pede identificação, chama billing.get_invoice + messaging.send_whatsapp → registra contexto + tool call.

## Plano de migração (fases)
1) **RAG MVP (um domínio, tenant piloto)**: 20–100 docs curados; filtros por tenant_id/valid_to; re-rank + threshold; citações.
2) **MCP para 3–5 tools críticas**: Encapar tooling atual; adicionar authz + logging; tools como lookup_customer, create_ticket, send_whatsapp, transfer_call, billing.get_invoice.
3) **Router + guardrails**: Classificador de intenção/risco; baixa confiança → clarificação ou humano; alto risco → políticas mais restritivas.
4) **Multi-tenant rígido**: Índices separados ou filtros mandatórios; credenciais/escopos por tenant no MCP; rate limits e quotas.
5) **Expandir domínios**: Mais MCPs (billing/crm/ticketing/telephony), mais namespaces de KB; adicionar reporting/export após ClickHouse populado.

## Ordem de implementação (serviços)
- **Primeiro**: Orquestrador/router + guardrails; serviço de RAG (API de retrieval) com filtros por tenant; esqueleto MCP com authz/logging.
- **Depois**: MCPs de domínio (billing, ticketing, CRM) integrando serviços existentes; MCP de telephony-control após fluxos de voz estáveis.
- **Python**: Processor de analytics quando eventos Kafka já tiverem tenant_id e schemas básicos; reporting/export depois de ClickHouse populado.

## Operação
- Fail closed em produção: se contexto ruim ou tool falhar, peça clarificação ou transfira; não invente.
- Rastrear IDs dos trechos recuperados por turn para explicabilidade e auditoria.
- Prompts enxutos; o “manual” vem do retrieval; exigir citações em system prompts.
- Versionar/deprecar docs com valid_to; usar ingestão incremental e reindex via Kafka.
