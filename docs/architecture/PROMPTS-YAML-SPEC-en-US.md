# prompts.yaml Specification (en-US)

## 1. Overview

`prompts.yaml` defines AI agents per tenant in Serphona. It captures routing, behavior, safety, tools, and now RAG/MCP settings (v1.1). Multi-tenant isolation is assumed (`tenant_id` present end-to-end).

## 2. Schema (v1.1, backward-compatible with 1.0)

```yaml
# prompts.yaml schema (v1.1)
version: "1.1"              # 1.1 adds RAG + MCP fields
tenant_id: "uuid"
default_agent: "agent-id"    # fallback agent when no routing match

# Global defaults (can be overridden per agent)
global:
  language: "pt-BR"
  tone: "professional"
  max_conversation_turns: 50
  conversation_timeout: "30m"
  rag_defaults:               # Optional shared retrieval defaults
    enabled: true
    top_k: 4
    rerank: true
    threshold: 0.35           # similarity threshold (0-1)
    namespaces: ["default"]  # logical KB namespaces (tenant-filtered)
    citations_required: true

agents:
  - id: "unique-agent-id"
    name: "Human-readable name"
    enabled: true
    role: "Agent role/persona"

    routing:
      tags: ["tag1", "tag2"]
      priority: 1              # higher tried first
      match_patterns:
        - "billing"
        - "invoice"

    behavior:
      system_prompt: |
        Multi-line system prompt
      personality:
        tone: "friendly"
        formality: "formal"
        verbosity: "concise"
      language_settings:
        primary: "pt-BR"
        supported: ["pt-BR", "en-US", "es-ES"]

    safety:
      rules:
        - "Never leak internal data"
      blocked_topics: ["politics", "religion"]
      pii_handling:
        mask_credit_cards: true
        mask_ssn: true
        mask_emails: false
      content_filtering:
        enabled: true
        level: "strict|moderate|permissive"

    knowledge:
      rag:                      # Per-agent retrieval config
        enabled: true
        namespaces: ["billing", "policies"]
        query:
          top_k: 4
          rerank: true
          threshold: 0.35
          filters:
            channel: "voice"
            language: "pt-BR"
          citations_required: true
          fallback: "ask_clarify"   # ask_clarify|fail_closed

    tools:                      # REST/gRPC/internal/MCP
      - name: "get_customer_info"
        type: "rest_api"       # rest_api|grpc|internal|mcp
        enabled: true
        config:
          endpoint: "https://api.example.com/customers/{customer_id}"
          method: "GET"
          auth_type: "bearer"
          timeout: "5s"
        schema:
          parameters:
            - name: "customer_id"
              type: "string"
              required: true
          response:
            type: "object"
            properties:
              name: "string"
              email: "string"
              status: "string"

      - name: "billing_get_invoice"
        type: "mcp"            # governed MCP tool
        enabled: true
        config:
          resource: "billing.get_invoice"  # MCP tool name
          timeout: "8s"
          idempotency_key: "conversation_id"
        schema:
          parameters:
            - name: "customer_id"
              type: "string"
              required: true
            - name: "invoice_id"
              type: "string"
              required: false
          response:
            type: "object"
            properties:
              invoice_id: "string"
              status: "string"
              pdf_url: "string"

    escalation:
      enabled: true
      conditions:
        - type: "sentiment"
          threshold: -0.7
          action: "transfer_human"
        - type: "conversation_turns"
          threshold: 10
          action: "offer_human"
        - type: "confidence"
          threshold: 0.3
          action: "offer_human"
      transfer_options:
        - type: "queue"
          queue_name: "support-queue"
          priority: 1

    fallback:
      no_understanding:
        - "Sorry, I didn't catch that. Could you rephrase?"
      error_occurred:
        - "Something went wrong. I'll transfer you to a human agent."
      timeout:
        - "Network seems slow. Please hold on."

    voice:
      stt_provider: "google"     # google|azure|aws|whisper
      tts_provider: "elevenlabs" # google|azure|aws|elevenlabs
      tts_voice_id: "voice-id"
      speech_rate: 1.0
      pitch: 0.0
      enable_interruptions: true
      silence_timeout: "3s"
```

## 3. RAG + MCP Extensions (v1.1)
- `global.rag_defaults`: shared retrieval defaults (top_k, rerank, threshold, namespaces, citations_required).
- `agents[].knowledge.rag`: per-agent retrieval config; enforce tenant and channel/language filters; for voice, prefer `fallback: fail_closed` when context quality is low.
- `agents[].tools[].type: mcp`: governed tools via MCP (`config.resource`, `timeout`, optional `idempotency_key`); schemas remain mandatory for inputs/outputs.
- System prompts should require citations when `citations_required: true`; responses should include snippet IDs for audit/explainability.
- Keep session memory separate from KB: RAG retrieves corporate docs; turn/session state stays in orchestrator/DB.

## 4. Guidance
- Multi-tenant safety: every retrieval and tool call must carry `tenant_id`; apply ACLs in services (RAG service, MCP server, downstream tools).
- Voice constraints: optimize for low latency; use low topK + rerank + similarity threshold; prefer fail-closed over hallucination.
- Tooling governance: use MCP for critical actions (billing, CRM, ticketing, telephony control); standardize error codes and idempotency keys.
- Observability: log retrieval IDs, tool calls (inputs/outputs hashed), durations, and decisions; trace end-to-end for audits and billing.
- Prompts slim by design: rely on RAG for knowledge; avoid embedding long manuals in prompts.
