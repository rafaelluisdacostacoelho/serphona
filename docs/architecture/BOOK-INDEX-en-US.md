# Architecture Book — Developer Path (en-US)

Purpose: a reading path so a new contributor can move from platform overview to service work with the right contracts and standards. Each step links to canonical docs.

## 0) Orientation
- [Architecture overview](README-en-US.md)
- [LIBS vs Services guidance](LIBS_VS_SERVICES-en-US.md)

## 1) Auth, envelopes, multi-tenancy
- [Auth guidance](AUTH-GUIDANCE-en-US.md)
- [Auth metrics & observability](AUTH-METRICS-OBSERVABILITY-en-US.md)
- [Tenant RLS guidance](TENANT-RLS-GUIDANCE-en-US.md)
- [Response envelope contract](RESPONSE-ENVELOPE-CONTRACT-en-US.md)

## 2) Data isolation & secrets
- [Tenant manager DB isolation](TENANT-MANAGER-DB-ISOLATION-en-US.md)
- [Per-service DB isolation playbook](SERVICE-DB-ISOLATION-en-US.md)
- [DB isolation for secrets](DB-ISOLATION-SECRETS-en-US.md)

## 3) RAG / MCP / Tools platform
- [RAG + MCP architecture](RAG-MCP-en-US.md)
- [Embedding strategy](EMBEDDING-platform-mcp-en-US.md)
- [Embed config](EMBED-CONFIG-platform-mcp-en-US.md)
- [Prompt schema](PROMPTS-YAML-SPEC-en-US.md)
- [Tools Gateway architecture](TOOLS-GATEWAY-ARCHITECTURE-en-US.md)
- [Tools Gateway config matrix](CONFIG-MATRIX-platform-mcp-en-US.md)
- [Tools Gateway embed config](EMBED-CONFIG-platform-mcp-en-US.md)
- [Rollout plan (platform-mcp)](ROLLOUT-platform-mcp-en-US.md)
- [RAG chunking strategy](RAG_CHUNKING_STRATEGY-en-US.md)

## 4) Voice and telephony
- [Voice gateway design](VOICE-GATEWAY-DESIGN-en-US.md)
- [Tenant telephony extensions](TENANT-MANAGER-TELEPHONY-EXTENSIONS-en-US.md)

## 5) Observability and events
- [Auth metrics & observability](AUTH-METRICS-OBSERVABILITY-en-US.md) (revisit for patterns)
- Platform observability (see service READMEs and dashboards)

## 6) Deployment & infra
- Helm/Terraform (see infra/helm, infra/terraform READMEs)
- Cost/rollout notes: refer to infra docs

## 7) Reference checklists
- [Testing phase notes](../testing-phase-notes-en-US.md)
- Service backlogs under `backend/go/services/*/BACKLOG.md` and libs under `backend/go/libs/*/BACKLOG.md`

## How to use
- Read in order; skim reference sections as needed.
- Keep a tab open for the service backlog you are implementing; it encodes the contract surface.
