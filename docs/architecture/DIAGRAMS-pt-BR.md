# Diagramas de Arquitetura (pt-BR)

## Visão Geral do Sistema
```mermaid
graph TD
  subgraph Frontend
    F1[console]
    F2[auth-mfe]
    F3[billing-mfe]
    F4[website]
  end

  subgraph Borda
    AG[auth-gateway]
  end

  subgraph PlanoDeControle
    TM[tenant-manager]
    TG[tools-gateway]
    TMgr[tools-manager]
    AO[agent-orchestrator]
    VG[voice-gateway]
  end

  subgraph PlanoDeDados
    AP[analytics-processor-service]
    RP[rag-processor-service]
    RE[reporting-export-service]
  end

  subgraph Libs
    PA[platform-auth]
    PMCP[platform-mcp]
    POBS[platform-observability]
    PRAG[platform-rag]
  end

  subgraph Dados
    PG[(Postgres + RLS)]
    CH[(ClickHouse)]
    KFK[(Kafka)]
    RED[(Redis)]
    MIN[(MinIO)]
  end

  subgraph Externos
    STT[Provider STT]
    TTS[Provider TTS]
    LLM[Provider LLM]
    Billing[Stripe/Payments]
  end

  F1 --> AG
  F2 --> AG
  F3 --> AG
  AG --> TM
  AG --> TG
  AG --> TMgr
  AG --> AO
  AG --> VG

  TM --> PG
  TMgr --> PG
  TG --> PG
  AO --> PG

  AO --> TG
  TG --> TMgr

  VG --> AO
  VG --> STT
  VG --> TTS

  AO --> LLM
  AO --> KFK
  AO --> RED

  AP --> CH
  RP --> CH
  RE --> PG
  RE --> MIN

  TG --> PMCP
  AO --> PMCP

  TM --> KFK
  TMgr --> KFK
  AP --> MIN
```

## Sequência — Chamada de Voz (STT → Agente → TTS)
```mermaid
sequenceDiagram
  participant Caller
  participant Asterisk
  participant VoiceGW as voice-gateway
  participant TenantMgr as tenant-manager
  participant Agent as agent-orchestrator
  participant STT
  participant LLM
  participant TTS

  Caller->>Asterisk: SIP INVITE
  Asterisk->>VoiceGW: canal ARI criado
  VoiceGW->>TenantMgr: GET tenant por DID (auth + envelope)
  TenantMgr-->>VoiceGW: tenant + config de provider
  VoiceGW->>STT: start stream (PCM 16k, escopo do tenant)
  VoiceGW->>Agent: POST /conversations (SERVICE_AUDIENCE)
  Agent-->>VoiceGW: conversation_id, metadados do agente
  loop Turns de fala
    STT-->>VoiceGW: transcrições parciais/finais
    VoiceGW->>Agent: POST /turns (tenant, trace, request IDs)
    Agent->>LLM: completion/plano de tool
    Agent-->>VoiceGW: texto de resposta + ações de tool
    VoiceGW->>TTS: sintetizar
    TTS-->>VoiceGW: áudio
    VoiceGW-->>Caller: reproduzir áudio
  end
  Caller-->>Asterisk: BYE
  VoiceGW->>Agent: POST /conversations/{id}/end
```

## Sequência — Resolução de Ferramenta via Tools Gateway / platform-mcp
```mermaid
sequenceDiagram
  participant Console
  participant AuthGW as auth-gateway
  participant ToolsGW as tools-gateway
  participant ToolsMgr as tools-manager
  participant MCP as platform-mcp (cliente)
  participant Agent as agent-orchestrator
  participant PG as Postgres

  Console->>AuthGW: GET /tools (JWT)
  AuthGW->>ToolsGW: forward com claims do tenant
  ToolsGW->>ToolsMgr: catálogo resolvido (ETag/If-None-Match)
  ToolsMgr-->>ToolsGW: tools + ETag (RLS)
  ToolsGW-->>AuthGW: tools
  AuthGW-->>Console: tools

  Note over Agent,MCP: Caminho de invocação de tool
  Agent->>MCP: invocar tool (tenant/request/trace headers)
  MCP->>ToolsGW: resolver schema/endpoint
  ToolsGW->>ToolsMgr: ler tool + versão
  ToolsMgr-->>ToolsGW: definição
  ToolsGW-->>MCP: endpoint/credenciais resolvidos
  MCP-->>Agent: resultado ou eventos streaming
```

## Diagrama de Classes — Catálogo de Ferramentas (simplificado)
```mermaid
classDiagram
  class Tool {
    uuid ID
    string Name
    string DisplayName
    string Description
    string[] Tags
    string? Category
    bool IsPublic
    bool IsDeprecated
  }
  class ToolVersion {
    uuid ID
    string Version
    string Status
    jsonb InputSchema
    jsonb OutputSchema
    jsonb Definition
    jsonb Allowlist
    jsonb Metadata
    int TimeoutSecs
    int MaxRetries
    int PayloadLimit
  }
  class TenantTool {
    uuid TenantID
    uuid ToolID
    uuid? ToolVersionID
    bool Enabled
    jsonb Policy
  }
  class PolicyRule {
    uuid ID
    string Effect
    string[] Scopes
    string[] Roles
    int Weight
    uuid? TenantID
  }
  class Secret {
    string ID
    uuid TenantID
    bytes ValueEncrypted
    timestamptz RotatedAt
  }

  Tool --> ToolVersion : tem muitos
  ToolVersion --> TenantTool : override opcional
  Tool --> TenantTool : habilitação por tenant
  TenantTool --> PolicyRule : avaliação de política
  Tool --> Secret : credenciais por tenant/tool
```
