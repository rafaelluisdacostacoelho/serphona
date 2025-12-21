# Backlog RAG & MCP (pt-BR)

## Objetivos
- Habilitar RAG multi-tenant com ingestao, indexacao e consulta segura.
- Expor ferramentas via MCP para desacoplar orquestracao de agentes das integracoes.
- Manter observabilidade, governanca e billing aderentes ao modelo atual (platform-auth, platform-observability, Tools Gateway).

## Principios
- Multi-tenant por design (tenant_id em todos os artefatos, RLS/ACL por namespace).
- Compatibilidade progressiva: MCP como fachada opcional, mantendo HTTP atual.
- Seguranca e conformidade primeiro: isolamento de dados, quotas, auditoria.
- Testes e rollout incremental por tenant/grupo de features.

## Assumcoes iniciais
- Armazenamento vetorial: avaliar pgvector (RLS) vs ClickHouse HNSW com particao por tenant_id.
- MinIO/S3 disponivel para blobs e metadados.
- Kafka habilitado para eventos de ingestao e reindexacao.
- platform-auth fornece identity com tenant_id e scopes para MCP e Tools Gateway.

## Epicos e Itens

### 1) Fundacao RAG
- [ ] Decidir vector store (pgvector vs ClickHouse) com benchmarks de latencia/custo e limites por tenant.
- [ ] Definir esquema de metadados: {tenant_id, namespace, document_id, version/etag, tags, acl, ttl}.
- [ ] Padronizar estrategia de chunking (tamanho, overlap, normalizacao de texto, idioma).
- [ ] Selecionar provider de embedding (OpenAI/Azure/OSS) com fallback e limites.
- [ ] Definir politicas de criptografia em repouso (KMS) e retenção/TTL por namespace.

### 2) Ingestao
- [ ] Criar contrato de evento Kafka "ingestion.requested" com tenant_id, source, document_id, etag.
- [ ] Implementar conector S3/MinIO (pull) com versionamento e deduplicacao por etag.
- [ ] Conector REST/GraphQL/gRPC para ingestao de datasets (paginacao, backoff, filtros por tenant).
- [ ] Conector SAP (OData/REST) com credenciais por tenant e masking de campos sensiveis.
- [ ] Registrar todas as ingestoes no Tools Gateway como ferramentas de ingestao (para audit/billing).

### 3) Pipeline de indexacao
- [ ] Worker Python (base analytics-processor-service) para: download blob, chunking, limpeza, embeddings, upsert no indice.
- [ ] Suporte a reprocessamento idempotente por document_id+etag.
- [ ] Publicar metricas (latencia por fase, tamanho medio do chunk, taxa de erro) em platform-observability.
- [ ] Cache de embeddings opcionais em Redis para reduzir custo em reindexacoes.
- [ ] Circuit breaker e DLQ para eventos com falha recorrente.

### 4) Consulta RAG
- [ ] Servico de retrieval com filtro por tenant_id, namespace, acl, score threshold e limitas por chamada.
- [ ] Re-ranker opcional (BM25+rerank) e dedup de trechos.
- [ ] Formatar contexto com citations (source, score, snippet) e conteudo seguro (masking).
- [ ] Cache de consultas quentes (Redis) com TTL curto e invalidacao por document_id.
- [ ] Observabilidade: trace end-to-end e logs de auditoria (consulta, score, top-k) para billing.

### 5) Integracao com Agent Orchestrator/Tools
- [ ] Estender spec `prompts.yaml` para `rag_sources`, `rag_query` (parametros: namespace, top_k, filters, rerank).
- [ ] Expor `rag_query` como tool no Tools Gateway com validacao de schema e billing por tokens retornados.
- [ ] Templates de prompt de sistem com regras de citacoes e limites de contexto por canal (voz/texto).
- [ ] Hook de guardrail para checar ACL de fonte antes de montar contexto.
- [ ] Testes de contrato (golden prompts) para regressao de comportamento.

### 6) MCP
- [ ] Implementar MCP server fino com descoberta de recursos do catalogo do Tools Gateway.
- [ ] Middleware de auth MCP usando platform-auth (token com tenant_id e scopes por recurso).
- [ ] Mapear tipos de provider: REST, GraphQL, gRPC, S3, `rag_query`.
- [ ] Observabilidade MCP: tracing, metricas por recurso, logs para reconciliar com billing.
- [ ] Client shim no Agent Orchestrator para resolver ferramentas via MCP com fallback HTTP.

### 7) Seguranca e Governanca
- [ ] Policy de quota e rate-limit por tenant/namespace para ingestao e consulta.
- [ ] Mascaramento de PII (e-mail, CPF/cartao) em ingestao e resposta de contexto.
- [ ] Controles de acesso por namespace (owner, readers, service-accounts) e scopes MCP.
- [ ] Revisao de secrets/credenciais: Vault/K8s Secrets, rotacao e audit trail.
- [ ] Playbook de incidentes para vazamento de dados e revogacao de acessos MCP.

### 8) Observabilidade e Billing
- [ ] Dashboards: ingestao (lag, throughput, erro), indexacao (latencia por fase), consulta (p95/p99, hit-rate cache).
- [ ] Alertas: latencia alta, erro de embedding provider, DLQ crescendo, quota estourada.
- [ ] Billing: custo por chamada de tool (ingestao/consulta), tokens de embedding, armazenamento vetorial.
- [ ] Logging de auditoria: quem consultou qual fonte, parametros de filtro, scores retornados.

### 9) Qualidade e Testes
- [ ] Suíte de testes unitarios e de contrato para ingestao, indexacao e consulta (offline).
- [ ] Testes de performance (latencia/throughput) com cargas por tenant e diferentes tamanhos de corpus.
- [ ] Testes de resiliencia (retry, backpressure, circuit breaker, DLQ).
- [ ] Testes de seguranca (ACL bypass, injeção em filtros, vazamento de contexto).

### 10) Rollout e Operacao
- [ ] Ligar por feature flag por tenant (ingestao, consulta, MCP separadamente).
- [ ] Plano de migracao: tenants piloto, monitorar e expandir.
- [ ] Runbooks: ingestao falhou, reindexacao, rotacao de chaves MCP, limpeza de namespace.
- [ ] Documentacao para times de suporte e clientes (onboarding de fontes, limites, exemplos de uso).

## Entregaveis
- Especificacao atualizada do `prompts.yaml` para RAG e MCP.
- MCP server funcional com discovery e auth multi-tenant.
- Pipeline de ingestao e indexacao operando com pelo menos uma fonte (S3/MinIO) e exemplo de consulta end-to-end.
- Dashboards e alertas basicos publicados.

## Riscos e Mitigacoes
- Custo de embeddings: usar cache, batch e provider fallback.
- Latencia de consulta: top-k ajustavel, cache e re-rank opcional.
- Vazamento de dados entre tenants: enforcement de tenant_id em todos os caminhos + testes de seguranca.
- Debt operacional: runbooks e feature flags para rollback rapido.
