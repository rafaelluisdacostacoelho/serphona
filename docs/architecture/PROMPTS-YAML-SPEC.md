# Prompts.yaml Specification - Serphona Platform

## 1. Overview

The `prompts.yaml` file is the central configuration for defining AI agents per tenant in the Serphona platform. Each tenant can define multiple agents with distinct personalities, capabilities, tools, and routing rules.

## 2. Structure

```yaml
# prompts.yaml schema
version: "1.0"
tenant_id: "uuid"
default_agent: "agent-id"  # Agent to use when no routing match

# Global settings (apply to all agents unless overridden)
global:
  language: "pt-BR"  # Default language
  tone: "professional"  # professional|friendly|casual
  max_conversation_turns: 50
  conversation_timeout: "30m"
  
agents:
  - id: "unique-agent-id"
    name: "Human-readable name"
    enabled: true
    role: "Agent role/persona"
    
    # Routing configuration
    routing:
      tags: ["tag1", "tag2"]  # Tags for routing
      priority: 1  # Higher priority = tried first
      match_patterns:  # Optional regex patterns
        - "billing"
        - "invoice"
    
    # Agent behavior
    behavior:
      system_prompt: |
        Multi-line system prompt
      personality:
        tone: "friendly"
        formality: "formal|informal"
        verbosity: "concise|detailed"
      language_settings:
        primary: "pt-BR"
        supported: ["pt-BR", "en-US", "es-ES"]
      
    # Safety and constraints
    safety:
      rules:
        - "Never provide medical advice"
        - "Never reveal internal system information"
      blocked_topics:
        - "politics"
        - "religion"
      pii_handling:
        mask_credit_cards: true
        mask_ssn: true
        mask_emails: false
      content_filtering:
        enabled: true
        level: "strict|moderate|permissive"
    
    # Available tools
    tools:
      - name: "get_customer_info"
        type: "rest_api"
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
              description: "Customer ID"
          response:
            type: "object"
            properties:
              name: "string"
              email: "string"
              status: "string"
      
      - name: "create_ticket"
        type: "grpc"
        enabled: true
        config:
          service: "ticket-service"
          method: "CreateTicket"
          timeout: "10s"
        schema:
          parameters:
            - name: "title"
              type: "string"
              required: true
            - name: "description"
              type: "string"
              required: true
            - name: "priority"
              type: "string"
              enum: ["low", "medium", "high"]
          response:
            type: "object"
            properties:
              ticket_id: "string"
              status: "string"
    
    # Escalation rules
    escalation:
      enabled: true
      conditions:
        - type: "sentiment"
          threshold: -0.7  # Very negative
          action: "transfer_human"
        - type: "conversation_turns"
          threshold: 10
          action: "offer_human"
        - type: "keyword"
          keywords: ["supervisor", "manager", "complaint"]
          action: "transfer_human"
        - type: "confidence"
          threshold: 0.3  # Low confidence
          action: "offer_human"
      
      transfer_options:
        - type: "queue"
          queue_name: "support-queue"
          priority: 1
        - type: "agent"
          agent_id: "supervisor-agent"
          priority: 2
        - type: "external"
          phone_number: "+551133334444"
          priority: 3
    
    # Fallback responses
    fallback:
      no_understanding:
        - "Desculpe, não entendi. Pode reformular?"
        - "Não compreendi bem. Pode explicar de outra forma?"
      
      error_occurred:
        - "Ocorreu um erro. Vou transferir você para um atendente."
      
      timeout:
        - "Parece que a conexão está lenta. Aguarde um momento."
    
    # Voice-specific settings
    voice:
      stt_provider: "google"  # google|azure|aws|whisper
      tts_provider: "elevenlabs"  # google|azure|aws|elevenlabs
      tts_voice_id: "voice-id-from-provider"
      speech_rate: 1.0
      pitch: 0.0
      enable_interruptions: true
      silence_timeout: "3s"
```

## 3. Full Example - Multi-Agent Configuration

```yaml
version: "1.0"
tenant_id: "550e8400-e29b-41d4-a716-446655440000"
default_agent: "receptionist-agent"

global:
  language: "pt-BR"
  tone: "professional"
  max_conversation_turns: 50
  conversation_timeout: "30m"

agents:
  # Agent 1: Receptionist (Initial contact point)
  - id: "receptionist-agent"
    name: "Recepcionista Virtual"
    enabled: true
    role: "Atendente de primeiro contato que identifica a necessidade do cliente e roteia para o agente especializado"
    
    routing:
      tags: ["reception", "initial", "triage"]
      priority: 1
      match_patterns:
        - ".*"  # Matches everything (default)
    
    behavior:
      system_prompt: |
        Você é um recepcionista virtual profissional e prestativo da empresa Acme Corp.
        Seu objetivo é cumprimentar o cliente, entender sua necessidade e direcioná-lo para o departamento correto.
        
        Sempre seja educado, conciso e eficiente.
        
        Após entender a necessidade do cliente, você deve:
        - Para questões de cobrança/financeiro: rotear para "billing-agent"
        - Para suporte técnico: rotear para "tech-support-agent"  
        - Para vendas: rotear para "sales-agent"
        - Para reclamações/SAC: rotear para humano
        
      personality:
        tone: "friendly"
        formality: "formal"
        verbosity: "concise"
      
      language_settings:
        primary: "pt-BR"
        supported: ["pt-BR", "en-US"]
    
    safety:
      rules:
        - "Nunca fornecer informações confidenciais da empresa"
        - "Nunca prometer soluções sem confirmar com departamento responsável"
      blocked_topics: []
      pii_handling:
        mask_credit_cards: true
        mask_ssn: true
        mask_emails: false
      content_filtering:
        enabled: true
        level: "moderate"
    
    tools:
      - name: "route_to_agent"
        type: "internal"
        enabled: true
        config:
          function: "routeConversation"
        schema:
          parameters:
            - name: "target_agent_id"
              type: "string"
              required: true
              enum: ["billing-agent", "tech-support-agent", "sales-agent"]
            - name: "reason"
              type: "string"
              required: true
    
    escalation:
      enabled: true
      conditions:
        - type: "keyword"
          keywords: ["reclamação", "ouvidoria", "advogado", "processo"]
          action: "transfer_human"
        - type: "sentiment"
          threshold: -0.8
          action: "transfer_human"
      
      transfer_options:
        - type: "queue"
          queue_name: "sac-queue"
          priority: 1
    
    fallback:
      no_understanding:
        - "Desculpe, não consegui entender. Você poderia repetir sua necessidade?"
        - "Não compreendi bem. Está buscando suporte técnico, informações sobre cobrança ou vendas?"
      
      error_occurred:
        - "Ocorreu um problema. Vou transferir você para um de nossos atendentes."
      
      timeout:
        - "A ligação está com problemas. Por favor, aguarde."
    
    voice:
      stt_provider: "google"
      tts_provider: "google"
      tts_voice_id: "pt-BR-Standard-A"
      speech_rate: 1.0
      pitch: 0.0
      enable_interruptions: true
      silence_timeout: "3s"

  # Agent 2: Billing Agent
  - id: "billing-agent"
    name: "Agente de Cobrança"
    enabled: true
    role: "Especialista em questões financeiras, faturas, pagamentos e segunda via de boletos"
    
    routing:
      tags: ["billing", "finance", "payment", "invoice"]
      priority: 2
      match_patterns:
        - "cobrança"
        - "fatura"
        - "boleto"
        - "pagamento"
        - "segunda via"
    
    behavior:
      system_prompt: |
        Você é um especialista em cobrança e finanças da Acme Corp.
        Você tem acesso aos sistemas financeiros e pode:
        - Consultar faturas e boletos
        - Enviar segunda via por email ou SMS
        - Verificar status de pagamentos
        - Negociar débitos (até 3 parcelas)
        - Informar sobre formas de pagamento
        
        Importante:
        - Sempre confirme os dados do cliente antes de fornecer informações financeiras
        - Para negociações acima de R$ 5.000, transfira para supervisor humano
        - Nunca processe pagamentos diretamente - apenas oriente o cliente
        
        Seja objetivo e claro nas informações financeiras.
        
      personality:
        tone: "professional"
        formality: "formal"
        verbosity: "detailed"
      
      language_settings:
        primary: "pt-BR"
        supported: ["pt-BR"]
    
    safety:
      rules:
        - "Sempre validar identidade do cliente antes de fornecer dados financeiros"
        - "Nunca solicitar senha ou CVV de cartão"
        - "Nunca processar pagamentos via telefone"
      blocked_topics:
        - "investment_advice"
      pii_handling:
        mask_credit_cards: true
        mask_ssn: true
        mask_emails: false
      content_filtering:
        enabled: true
        level: "strict"
    
    tools:
      - name: "get_customer_invoices"
        type: "rest_api"
        enabled: true
        config:
          endpoint: "https://api.acme.com/v1/customers/{customer_id}/invoices"
          method: "GET"
          auth_type: "bearer"
          timeout: "5s"
          retry_attempts: 3
        schema:
          parameters:
            - name: "customer_id"
              type: "string"
              required: true
            - name: "status"
              type: "string"
              enum: ["pending", "paid", "overdue"]
          response:
            type: "array"
            items:
              type: "object"
              properties:
                invoice_id: "string"
                amount: "number"
                due_date: "string"
                status: "string"
      
      - name: "send_invoice_copy"
        type: "rest_api"
        enabled: true
        config:
          endpoint: "https://api.acme.com/v1/invoices/{invoice_id}/send"
          method: "POST"
          auth_type: "bearer"
          timeout: "10s"
        schema:
          parameters:
            - name: "invoice_id"
              type: "string"
              required: true
            - name: "delivery_method"
              type: "string"
              enum: ["email", "sms"]
              required: true
          response:
            type: "object"
            properties:
              sent: "boolean"
              message: "string"
      
      - name: "check_payment_status"
        type: "rest_api"
        enabled: true
        config:
          endpoint: "https://api.acme.com/v1/payments/{payment_id}/status"
          method: "GET"
          auth_type: "bearer"
          timeout: "5s"
        schema:
          parameters:
            - name: "payment_id"
              type: "string"
              required: true
          response:
            type: "object"
            properties:
              status: "string"
              paid_at: "string"
              amount: "number"
    
    escalation:
      enabled: true
      conditions:
        - type: "custom"
          condition: "amount > 5000"
          action: "transfer_human"
        - type: "keyword"
          keywords: ["fraude", "contestação", "advogado"]
          action: "transfer_human"
        - type: "conversation_turns"
          threshold: 15
          action: "offer_human"
      
      transfer_options:
        - type: "queue"
          queue_name: "billing-supervisor-queue"
          priority: 1
        - type: "agent"
          agent_id: "supervisor-agent"
          priority: 2
    
    fallback:
      no_understanding:
        - "Não entendi sua solicitação financeira. Você está buscando informações sobre fatura, boleto ou pagamento?"
      
      error_occurred:
        - "Não consegui acessar os dados financeiros no momento. Vou transferir para um atendente humano."
      
      timeout:
        - "O sistema financeiro está demorando para responder. Por favor, aguarde."
    
    voice:
      stt_provider: "google"
      tts_provider: "elevenlabs"
      tts_voice_id: "21m00Tcm4TlvDq8ikWAM"  # Professional male voice
      speech_rate: 0.95
      pitch: 0.0
      enable_interruptions: false  # Don't interrupt during financial info
      silence_timeout: "4s"

  # Agent 3: Technical Support Agent
  - id: "tech-support-agent"
    name: "Agente de Suporte Técnico"
    enabled: true
    role: "Especialista em suporte técnico, troubleshooting e resolução de problemas com produtos/serviços"
    
    routing:
      tags: ["technical", "support", "troubleshooting", "problem"]
      priority: 2
      match_patterns:
        - "não funciona"
        - "problema"
        - "erro"
        - "bug"
        - "suporte técnico"
        - "ajuda técnica"
    
    behavior:
      system_prompt: |
        Você é um especialista em suporte técnico da Acme Corp.
        Você tem conhecimento profundo sobre todos os produtos e serviços da empresa.
        
        Sua abordagem deve ser:
        1. Diagnosticar o problema com perguntas específicas
        2. Tentar resolver remotamente primeiro
        3. Se não resolver, criar ticket de suporte
        4. Se urgente, escalar para técnico humano
        
        Você tem acesso a:
        - Base de conhecimento com soluções comuns
        - Sistema de tickets
        - Status de servidores e serviços
        - Histórico de atendimentos do cliente
        
        Seja paciente e didático, especialmente com clientes menos técnicos.
        
      personality:
        tone: "friendly"
        formality: "informal"
        verbosity: "detailed"
      
      language_settings:
        primary: "pt-BR"
        supported: ["pt-BR", "en-US"]
    
    safety:
      rules:
        - "Nunca solicitar senhas ou dados de login"
        - "Nunca executar comandos sem explicar ao cliente"
        - "Sempre validar impacto antes de sugerir ações destrutivas"
      blocked_topics: []
      pii_handling:
        mask_credit_cards: true
        mask_ssn: true
        mask_emails: false
      content_filtering:
        enabled: true
        level: "moderate"
    
    tools:
      - name: "search_knowledge_base"
        type: "rest_api"
        enabled: true
        config:
          endpoint: "https://api.acme.com/v1/kb/search"
          method: "POST"
          auth_type: "bearer"
          timeout: "5s"
        schema:
          parameters:
            - name: "query"
              type: "string"
              required: true
            - name: "product"
              type: "string"
              required: false
          response:
            type: "array"
            items:
              type: "object"
              properties:
                article_id: "string"
                title: "string"
                solution: "string"
                relevance: "number"
      
      - name: "create_support_ticket"
        type: "grpc"
        enabled: true
        config:
          service: "support-service"
          method: "CreateTicket"
          timeout: "10s"
        schema:
          parameters:
            - name: "customer_id"
              type: "string"
              required: true
            - name: "title"
              type: "string"
              required: true
            - name: "description"
              type: "string"
              required: true
            - name: "priority"
              type: "string"
              enum: ["low", "medium", "high", "critical"]
            - name: "category"
              type: "string"
              enum: ["bug", "feature_request", "question", "incident"]
          response:
            type: "object"
            properties:
              ticket_id: "string"
              status: "string"
              assigned_to: "string"
      
      - name: "check_service_status"
        type: "rest_api"
        enabled: true
        config:
          endpoint: "https://status.acme.com/api/v1/status"
          method: "GET"
          auth_type: "none"
          timeout: "3s"
        schema:
          parameters: []
          response:
            type: "object"
            properties:
              services:
                type: "array"
                items:
                  type: "object"
                  properties:
                    name: "string"
                    status: "string"
                    last_incident: "string"
      
      - name: "get_customer_history"
        type: "rest_api"
        enabled: true
        config:
          endpoint: "https://api.acme.com/v1/customers/{customer_id}/support-history"
          method: "GET"
          auth_type: "bearer"
          timeout: "5s"
        schema:
          parameters:
            - name: "customer_id"
              type: "string"
              required: true
            - name: "limit"
              type: "integer"
              default: 10
          response:
            type: "array"
            items:
              type: "object"
              properties:
                ticket_id: "string"
                created_at: "string"
                issue: "string"
                resolution: "string"
    
    escalation:
      enabled: true
      conditions:
        - type: "keyword"
          keywords: ["urgente", "crítico", "parado", "prejuízo"]
          action: "transfer_human"
        - type: "conversation_turns"
          threshold: 20
          action: "offer_human"
        - type: "custom"
          condition: "unresolved_after_3_solutions"
          action: "transfer_human"
      
      transfer_options:
        - type: "queue"
          queue_name: "l2-support-queue"
          priority: 1
        - type: "queue"
          queue_name: "emergency-support"
          priority: 2
          condition: "critical_issue"
    
    fallback:
      no_understanding:
        - "Não compreendi bem o problema. Pode descrever o que está acontecendo com mais detalhes?"
        - "Desculpe, não entendi. Qual produto ou serviço está apresentando problema?"
      
      error_occurred:
        - "Encontrei um erro ao buscar a solução. Vou criar um ticket e transferir para a equipe técnica."
      
      timeout:
        - "O sistema está demorando para responder. Deixe-me criar um ticket para você."
    
    voice:
      stt_provider: "google"
      tts_provider: "elevenlabs"
      tts_voice_id: "pNInz6obpgDQGcFmaJgB"  # Calm and patient voice
      speech_rate: 0.9  # Slightly slower for technical instructions
      pitch: 0.0
      enable_interruptions: true
      silence_timeout: "4s"

  # Agent 4: Sales Agent
  - id: "sales-agent"
    name: "Agente de Vendas"
    enabled: true
    role: "Consultor de vendas focado em entender necessidades e oferecer soluções adequadas"
    
    routing:
      tags: ["sales", "commercial", "purchase", "upgrade"]
      priority: 2
      match_patterns:
        - "comprar"
        - "contratar"
        - "plano"
        - "preço"
        - "upgrade"
        - "vendas"
    
    behavior:
      system_prompt: |
        Você é um consultor de vendas consultivo e orientado a valor da Acme Corp.
        Sua abordagem é NUNCA ser agressivo ou insistente.
        
        Seu processo:
        1. Entender a necessidade do cliente com perguntas abertas
        2. Identificar dores e objetivos
        3. Apresentar a solução mais adequada (não a mais cara)
        4. Explicar benefícios focando no valor, não no preço
        5. Se não houver fit, ser honesto e não forçar venda
        
        Você conhece todos os produtos, planos, preços e promoções atuais.
        
        Regras importantes:
        - Nunca oferecer descontos não autorizados (máximo 10%)
        - Para vendas corporativas (>10 licenças), transferir para comercial humano
        - Sempre confirmar entendimento antes de fechar
        - Ser transparente sobre termos e condições
        
      personality:
        tone: "friendly"
        formality: "informal"
        verbosity: "detailed"
      
      language_settings:
        primary: "pt-BR"
        supported: ["pt-BR", "en-US", "es-ES"]
    
    safety:
      rules:
        - "Nunca fazer promessas não verificáveis"
        - "Sempre mencionar período de trial quando disponível"
        - "Ser transparente sobre política de cancelamento"
      blocked_topics: []
      pii_handling:
        mask_credit_cards: true
        mask_ssn: true
        mask_emails: false
      content_filtering:
        enabled: true
        level: "moderate"
    
    tools:
      - name: "get_product_catalog"
        type: "rest_api"
        enabled: true
        config:
          endpoint: "https://api.acme.com/v1/catalog"
          method: "GET"
          auth_type: "bearer"
          timeout: "5s"
        schema:
          parameters:
            - name: "category"
              type: "string"
              required: false
          response:
            type: "array"
            items:
              type: "object"
              properties:
                product_id: "string"
                name: "string"
                description: "string"
                price: "number"
                features: "array"
      
      - name: "check_promotions"
        type: "rest_api"
        enabled: true
        config:
          endpoint: "https://api.acme.com/v1/promotions/active"
          method: "GET"
          auth_type: "bearer"
          timeout: "3s"
        schema:
          parameters: []
          response:
            type: "array"
            items:
              type: "object"
              properties:
                promo_id: "string"
                name: "string"
                discount_percent: "number"
                applicable_products: "array"
                valid_until: "string"
      
      - name: "create_quote"
        type: "rest_api"
        enabled: true
        config:
          endpoint: "https://api.acme.com/v1/quotes"
          method: "POST"
          auth_type: "bearer"
          timeout: "8s"
        schema:
          parameters:
            - name: "customer_id"
              type: "string"
              required: true
            - name: "products"
              type: "array"
              required: true
            - name: "discount_code"
              type: "string"
              required: false
          response:
            type: "object"
            properties:
              quote_id: "string"
              total_amount: "number"
              valid_until: "string"
              payment_link: "string"
      
      - name: "schedule_demo"
        type: "rest_api"
        enabled: true
        config:
          endpoint: "https://api.acme.com/v1/demos/schedule"
          method: "POST"
          auth_type: "bearer"
          timeout: "5s"
        schema:
          parameters:
            - name: "customer_name"
              type: "string"
              required: true
            - name: "email"
              type: "string"
              required: true
            - name: "phone"
              type: "string"
              required: true
            - name: "preferred_date"
              type: "string"
              required: false
          response:
            type: "object"
            properties:
              demo_id: "string"
              scheduled_at: "string"
              meeting_link: "string"
    
    escalation:
      enabled: true
      conditions:
        - type: "custom"
          condition: "corporate_sale"  # >10 licenses
          action: "transfer_human"
        - type: "keyword"
          keywords: ["contrato customizado", "enterprise", "licitação"]
          action: "transfer_human"
        - type: "conversation_turns"
          threshold: 25
          action: "offer_human"
      
      transfer_options:
        - type: "queue"
          queue_name: "sales-consultants"
          priority: 1
        - type: "queue"
          queue_name: "enterprise-sales"
          priority: 2
          condition: "corporate_sale"
    
    fallback:
      no_understanding:
        - "Não entendi bem sua necessidade. Está buscando informações sobre qual produto ou serviço?"
      
      error_occurred:
        - "Não consegui acessar o catálogo no momento. Deixe-me transferir para um consultor de vendas."
      
      timeout:
        - "O sistema está lento. Posso agendar uma conversa com nosso time comercial?"
    
    voice:
      stt_provider: "google"
      tts_provider: "elevenlabs"
      tts_voice_id: "EXAVITQu4vr4xnSDxMaL"  # Friendly and enthusiastic
      speech_rate: 1.05  # Slightly faster, energetic
      pitch: 0.5  # Slightly higher pitch
      enable_interruptions: true
      silence_timeout: "3s"
```

## 4. Storage and Management

### Option 1: Database Storage (Recommended)
- Store in PostgreSQL in tenant-manager database
- Table: `agent_configurations`
- Fields: tenant_id, config_yaml, version, updated_at
- Cache in Redis with 1-hour TTL
- API endpoints for CRUD operations

### Option 2: Object Storage (S3/MinIO)
- Path: `s3://serphona-configs/{tenant_id}/prompts.yaml`
- Versioned for rollback capability
- Cached in Redis

### Option 3: Git Repository
- Separate repo per tenant or mono-repo with folders
- CI/CD pipeline validates and deploys changes
- Best for tenants who want version control

## 5. Validation Schema

The platform should validate prompts.yaml against JSON Schema:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["version", "tenant_id", "agents"],
  "properties": {
    "version": {"type": "string"},
    "tenant_id": {"type": "string", "format": "uuid"},
    "default_agent": {"type": "string"},
    "agents": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["id", "name", "enabled", "role", "behavior"],
        "properties": {
          "id": {"type": "string"},
          "name": {"type": "string"},
          "enabled": {"type": "boolean"},
          "tools": {
            "type": "array",
            "items": {
              "required": ["name", "type", "enabled"],
              "properties": {
                "name": {"type": "string"},
                "type": {"enum": ["rest_api", "grpc", "internal"]}
              }
            }
          }
        }
      }
    }
  }
}
```

## 6. Runtime Usage

### In agent-orchestrator:
```go
// Load agent configuration
config := agentConfig.LoadForTenant(ctx, tenantID)
agent := config.GetAgentByID(agentID)

// Build prompt
prompt := agent.BuildPrompt(conversationHistory)

// Check if tool is allowed
if agent.IsToolAllowed("create_ticket") {
    // Execute tool via tools-gateway
}

// Check safety rules
if agent.ViolatesSafetyRules(userMessage) {
    return agent.GetFallbackResponse("blocked_content")
}
```

### In voice-gateway:
```go
// Determine which agent to use based on routing
agent := routingEngine.SelectAgent(
    userMessage, 
    conversationContext,
    tenantConfig,
)

// Get voice settings
voiceConfig := agent.Voice
sttProvider := sttFactory.Get(voiceConfig.STTProvider)
ttsProvider := ttsFactory.Get(voiceConfig.TTSProvider)
```
