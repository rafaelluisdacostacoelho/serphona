# Livro de Arquitetura — Trilha para Dev (pt-BR)

Propósito: uma trilha de leitura para que quem chega entenda a plataforma, contratos e padrões antes de codar. Cada etapa aponta para os docs canônicos.

## 0) Orientação
- [Visão geral de arquitetura](README-pt-BR.md)
- [Guia LIBS vs Services](LIBS_VS_SERVICES-pt-BR.md)

## 1) Auth, envelopes e multi-tenancy
- [Guia de autenticação](AUTH-GUIDANCE-pt-BR.md)
- [Métricas/observabilidade de auth](AUTH-METRICS-OBSERVABILITY-pt-BR.md)
- [Guia de RLS por tenant](TENANT-RLS-GUIDANCE-pt-BR.md)
- [Contrato de envelope de resposta](RESPONSE-ENVELOPE-CONTRACT-pt-BR.md)

## 2) Isolamento de dados e segredos
- [Isolamento de DB no tenant-manager](TENANT-MANAGER-DB-ISOLATION-pt-BR.md)
- [Playbook de isolamento de DB por serviço](SERVICE-DB-ISOLATION-pt-BR.md)
- [Isolamento de DB para segredos](DB-ISOLATION-SECRETS-pt-BR.md)

## 3) RAG / MCP / Tools
- [Arquitetura RAG + MCP](RAG-MCP-pt-BR.md)
- [Estratégia de embedding](EMBEDDING-platform-mcp-pt-BR.md)
- [Config de embedding](EMBED-CONFIG-platform-mcp-pt-BR.md)
- [Esquema de prompts](PROMPTS-YAML-SPEC-pt-BR.md)
- [Arquitetura do Tools Gateway](TOOLS-GATEWAY-ARCHITECTURE-pt-BR.md)
- [Matriz de config do platform-mcp](CONFIG-MATRIX-platform-mcp-pt-BR.md)
- [Config de embed do platform-mcp](EMBED-CONFIG-platform-mcp-pt-BR.md)
- [Plano de rollout (platform-mcp)](ROLLOUT-platform-mcp-pt-BR.md)
- [Estratégia de chunking RAG](RAG_CHUNKING_STRATEGY-pt-BR.md)

## 4) Voz e telefonia
- [Design do voice gateway](VOICE-GATEWAY-DESIGN-pt-BR.md)
- [Extensões de telefonia no tenant-manager](TENANT-MANAGER-TELEPHONY-EXTENSIONS-pt-BR.md)

## 5) Observabilidade e eventos
- [Métricas/observabilidade de auth](AUTH-METRICS-OBSERVABILITY-pt-BR.md) (revisite para padrões)
- Observabilidade geral: ver READMEs dos serviços e dashboards

## 6) Deploy e infra
- Helm/Terraform: ver READMEs em `infra/helm` e `infra/terraform`
- Custos/rollout: ver docs em infra

## 7) Checklists de referência
- [Notas da fase de testes](../testing-phase-notes-pt-BR.md)
- Backlogs de serviços em `backend/go/services/*/BACKLOG.md` e libs em `backend/go/libs/*/BACKLOG.md`

## Como usar
- Leia na ordem; consulte referências conforme necessário.
- Mantenha aberto o backlog do serviço que está implementando; ele define o contrato.
