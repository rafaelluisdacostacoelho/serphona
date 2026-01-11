# Architecture Diagrams (en-US)

## System Overview
```mermaid
graph TD
  subgraph Frontend
    F1[console]
    F2[auth-mfe]
    F3[billing-mfe]
    F4[website]
  end

  subgraph Edge
    AG[auth-gateway]
  end

  subgraph ControlPlane
    TM[tenant-manager]
    TG[tools-gateway]
    TMgr[tools-manager]
    AO[agent-orchestrator]
    VG[voice-gateway]
  end

  subgraph DataPlane
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

  subgraph DataStores
    PG[(Postgres + RLS)]
    CH[(ClickHouse)]
    KFK[(Kafka)]
    RED[(Redis)]
    MIN[(MinIO)]
  end

  subgraph External
    STT[STT provider]
    TTS[TTS provider]
    LLM[LLM provider]
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

## Sequence — Voice Call (STT → Agent → TTS)
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
  Asterisk->>VoiceGW: ARI channel created
  VoiceGW->>TenantMgr: GET tenant by DID (auth + envelope)
  TenantMgr-->>VoiceGW: tenant + provider config
  VoiceGW->>STT: start stream (PCM 16k, tenant-scoped)
  VoiceGW->>Agent: POST /conversations (SERVICE_AUDIENCE token)
  Agent-->>VoiceGW: conversation_id, agent metadata
  loop Speech turns
    STT-->>VoiceGW: partial + final transcripts
    VoiceGW->>Agent: POST /turns (tenant, trace, request IDs)
    Agent->>LLM: completion/tool plan
    Agent-->>VoiceGW: reply text + tool actions (if any)
    VoiceGW->>TTS: synthesize reply
    TTS-->>VoiceGW: audio
    VoiceGW-->>Caller: play audio
  end
  Caller-->>Asterisk: BYE
  VoiceGW->>Agent: POST /conversations/{id}/end
```

## Sequence — Tool Resolution via Tools Gateway / platform-mcp
```mermaid
sequenceDiagram
  participant Console
  participant AuthGW as auth-gateway
  participant ToolsGW as tools-gateway
  participant ToolsMgr as tools-manager
  participant MCP as platform-mcp (client)
  participant Agent as agent-orchestrator
  participant PG as Postgres

  Console->>AuthGW: GET /tools (JWT)
  AuthGW->>ToolsGW: forward with tenant claims
  ToolsGW->>ToolsMgr: fetch resolved catalog (ETag/If-None-Match)
  ToolsMgr-->>ToolsGW: tools + ETag (RLS enforced)
  ToolsGW-->>AuthGW: tools
  AuthGW-->>Console: tools

  Note over Agent,MCP: Tool invocation path
  Agent->>MCP: invoke tool (tenant/request/trace headers)
  MCP->>ToolsGW: resolve tool schema/endpoint
  ToolsGW->>ToolsMgr: read tool + version
  ToolsMgr-->>ToolsGW: tool definition
  ToolsGW-->>MCP: resolved endpoint/credentials
  MCP-->>Agent: result or streaming events
```

## Class Diagram — Tools Catalog (simplified)
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

  Tool --> ToolVersion : has many
  ToolVersion --> TenantTool : optional override
  Tool --> TenantTool : enablement per tenant
  TenantTool --> PolicyRule : evaluated via tenant policy
  Tool --> Secret : credentials per tenant/tool
```
