# Contrato de Evento de Uso (usage.reported)

Propósito: alinhar emissores (tenant-manager e demais serviços) com o consumidor billing-service para uso, quotas e excedentes.

## Tópico e Roteamento
- Tópico Kafka: `usage.reported`
- Chave: `tenant_id` (UUID em string); garante ordenação por tenant
- Valor: JSON (abaixo); Avro/Protobuf são aceitáveis futuramente com schema registry

## Payload
| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| tenant_id | string (UUID) | sim | Identificador do tenant; também chave de partição |
| period | string (YYYY-MM) | sim | Janela mensal (UTC) |
| occurred_at | string (RFC3339) | sim | Momento em que o uso ocorreu (UTC) |
| source | string | sim | Serviço que gerou o uso (ex.: `tenant-manager`, `voice-gateway`, `analytics-processor`) |
| calls | integer >= 0 | sim | Delta de chamadas |
| minutes | integer >= 0 | sim | Delta de minutos |
| messages | integer >= 0 | sim | Delta de mensagens (se aplicável) |
| storage_gb | number >= 0 | sim | Delta de armazenamento em GB (pode ser fracionário) |
| api_requests | integer >= 0 | sim | Delta de requisições de API |
| plan | string | não | Plano vigente no momento do uso |
| subscription_id | string | não | ID da assinatura no billing (se existir) |
| request_id | string | sim | Chave de idempotência deste registro de uso |
| trace_id | string | não | Correlacionar tracing |

Notas:
- Todos os contadores são **deltas**, nunca totais.
- Emita um evento por operação que consome quota; deduplique por `request_id`.
- Nunca envie deltas negativos.

## Exemplo
```json
{
  "tenant_id": "9c1c3e79-7e5d-4e30-8a6e-4f3bb7cb4c0f",
  "period": "2025-12",
  "occurred_at": "2025-12-31T23:59:12Z",
  "source": "tenant-manager",
  "calls": 2,
  "minutes": 6,
  "messages": 0,
  "storage_gb": 0.0,
  "api_requests": 3,
  "plan": "starter",
  "subscription_id": "sub_123",
  "request_id": "req-7f5f8d1e-321e-4c71-9a3f-62b6f0e9c111",
  "trace_id": "2a4b5c6d7e8f9a0b"
}
```

## Semântica de Reset/Período
- Janelas são mensais (`period=YYYY-MM`, UTC).
- O reset de quota ocorre na virada do período; produtores devem definir `period` pelo horário de ocorrência, não de envio.

## Regras de Validação (esperadas pelo consumidor)
- Rejeitar/ignorar mensagens sem `tenant_id`, `period`, `occurred_at` ou com deltas negativos.
- Idempotência: `request_id` duplicado para o mesmo `tenant_id` deve ser no-op.

## Perguntas em aberto
- `messages`/`api_requests` devem ser opcionais por plano? (por padrão zero permitido)
- Precisamos de dimensões por produto (ex.: `transcription_seconds`, `storage_write_gb`)?
- Excedentes são cobrados imediatamente por mensagem ou em lote por período?
