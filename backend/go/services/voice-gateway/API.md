# Voice Gateway - API Documentation

Documentação completa da API REST do Voice Gateway.

## 📋 Índice

- [Autenticação](#autenticação)
- [Endpoints](#endpoints)
  - [Health Checks](#health-checks)
  - [Call Management](#call-management)
  - [Asterisk Webhooks](#asterisk-webhooks)
- [Modelos de Dados](#modelos-de-dados)
- [Códigos de Status](#códigos-de-status)
- [Exemplos](#exemplos)

## 🔐 Autenticação

A maioria dos endpoints requer autenticação via JWT token.

```http
Authorization: Bearer <jwt_token>
```

## 📡 Endpoints

### Base URL

```
Development: http://localhost:8080
Production: https://voice-gateway.serphona.com
```

### Health Checks

#### GET /health

Verifica o status geral do serviço.

**Response**

```json
{
  "status": "healthy",
  "service": "voice-gateway"
}
```

**Status Codes**
- `200 OK` - Serviço saudável

---

#### GET /health/live

Liveness probe para Kubernetes.

**Response**

```json
{
  "status": "alive"
}
```

**Status Codes**
- `200 OK` - Serviço vivo

---

#### GET /health/ready

Readiness probe para Kubernetes.

**Response**

```json
{
  "status": "ready"
}
```

**Status Codes**
- `200 OK` - Serviço pronto para receber tráfego
- `503 Service Unavailable` - Serviço não está pronto

---

### Call Management

#### GET /api/v1/calls/{call_id}

Obtém informações de uma chamada específica.

**Parameters**

| Nome | Tipo | Localização | Descrição |
|------|------|-------------|-----------|
| `call_id` | UUID | Path | ID da chamada |

**Response**

```json
{
  "call_id": "123e4567-e89b-12d3-a456-426614174000",
  "tenant_id": "987fcdeb-51a2-43d7-8f9e-123456789abc",
  "direction": "inbound",
  "caller_number": "+5511999887766",
  "callee_number": "+5511988776655",
  "state": "active",
  "conversation_id": "456e7890-a12b-34c5-d678-901234567def",
  "agent_id": "agent-receptionist",
  "duration_ms": 45000,
  "metadata": {
    "customer_name": "John Doe",
    "campaign_id": "summer-2024"
  }
}
```

**Status Codes**
- `200 OK` - Chamada encontrada
- `400 Bad Request` - call_id inválido
- `404 Not Found` - Chamada não encontrada
- `401 Unauthorized` - Token inválido

---

#### DELETE /api/v1/calls/{call_id}

Encerra uma chamada.

**Parameters**

| Nome | Tipo | Localização | Descrição |
|------|------|-------------|-----------|
| `call_id` | UUID | Path | ID da chamada |

**Request Body (opcional)**

```json
{
  "reason": "user_hangup"
}
```

**Response**

```json
{
  "status": "call ended"
}
```

**Status Codes**
- `200 OK` - Chamada encerrada
- `400 Bad Request` - call_id inválido
- `404 Not Found` - Chamada não encontrada
- `500 Internal Server Error` - Erro ao encerrar

---

#### POST /api/v1/calls/{call_id}/transfer

Transfere uma chamada para fila, agente ou número externo.

**Parameters**

| Nome | Tipo | Localização | Descrição |
|------|------|-------------|-----------|
| `call_id` | UUID | Path | ID da chamada |

**Request Body**

```json
{
  "type": "queue",
  "target": "support-queue",
  "reason": "escalation"
}
```

**Tipos de Transfer**

| Tipo | Target | Descrição |
|------|--------|-----------|
| `queue` | Nome da fila | Transferir para fila ACD |
| `agent` | ID do agente | Transferir para agente específico |
| `external` | Número E.164 | Transferir para número externo |

**Response**

```json
{
  "status": "call transferred"
}
```

**Status Codes**
- `200 OK` - Chamada transferida
- `400 Bad Request` - Parâmetros inválidos
- `404 Not Found` - Chamada não encontrada
- `500 Internal Server Error` - Erro na transferência

---

#### GET /api/v1/tenants/{tenant_id}/calls

Lista chamadas ativas de um tenant.

**Parameters**

| Nome | Tipo | Localização | Descrição |
|------|------|-------------|-----------|
| `tenant_id` | UUID | Path | ID do tenant |

**Query Parameters**

| Nome | Tipo | Padrão | Descrição |
|------|------|--------|-----------|
| `state` | string | - | Filtrar por estado |
| `direction` | string | - | Filtrar por direção |
| `limit` | int | 50 | Número máximo de resultados |
| `offset` | int | 0 | Offset para paginação |

**Response**

```json
{
  "calls": [
    {
      "call_id": "123e4567-e89b-12d3-a456-426614174000",
      "tenant_id": "987fcdeb-51a2-43d7-8f9e-123456789abc",
      "direction": "inbound",
      "caller_number": "+5511999887766",
      "callee_number": "+5511988776655",
      "state": "active",
      "agent_id": "agent-support",
      "metadata": {}
    },
    {
      "call_id": "234f5678-f90c-23e4-b567-537725285111",
      "tenant_id": "987fcdeb-51a2-43d7-8f9e-123456789abc",
      "direction": "outbound",
      "caller_number": "+5511988776655",
      "callee_number": "+5511977665544",
      "state": "ringing",
      "agent_id": "agent-sales",
      "metadata": {}
    }
  ],
  "total": 2
}
```

**Status Codes**
- `200 OK` - Lista retornada
- `400 Bad Request` - tenant_id inválido
- `401 Unauthorized` - Token inválido
- `500 Internal Server Error` - Erro interno

---

### Asterisk Webhooks

#### POST /asterisk/events

Recebe eventos do Asterisk ARI via webhook.

**⚠️ Este endpoint é chamado pelo Asterisk, não por clientes externos**

**Request Body**

```json
{
  "type": "StasisStart",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "channel": {
    "id": "PJSIP/trunk-00000001",
    "name": "PJSIP/trunk-00000001",
    "state": "Ring",
    "caller": {
      "number": "+5511999887766",
      "name": "John Doe"
    },
    "connected": {
      "number": "+5511988776655",
      "name": ""
    }
  }
}
```

**Tipos de Eventos Suportados**

| Tipo | Descrição |
|------|-----------|
| `StasisStart` | Canal entrou na aplicação Stasis |
| `StasisEnd` | Canal saiu da aplicação Stasis |
| `ChannelAnswered` | Canal foi atendido |
| `ChannelHangupRequest` | Pedido de desligamento |
| `ChannelDestroyed` | Canal foi destruído |

**Response**

```json
{}
```

**Status Codes**
- `200 OK` - Evento processado
- `400 Bad Request` - Evento inválido

---

## 📦 Modelos de Dados

### Call

```typescript
interface Call {
  call_id: string;          // UUID
  tenant_id: string;        // UUID
  direction: "inbound" | "outbound";
  caller_number: string;    // E.164 format
  callee_number: string;    // E.164 format
  state: CallState;
  conversation_id?: string; // UUID
  agent_id?: string;
  duration_ms?: number;
  metadata?: Record<string, any>;
}
```

### CallState

```typescript
type CallState = 
  | "ringing"
  | "answered"
  | "active"
  | "hold"
  | "transferred"
  | "ended"
  | "error";
```

### Direction

```typescript
type Direction = "inbound" | "outbound";
```

---

## 📊 Códigos de Status

| Código | Descrição |
|--------|-----------|
| `200 OK` | Requisição bem-sucedida |
| `201 Created` | Recurso criado |
| `204 No Content` | Sucesso sem corpo de resposta |
| `400 Bad Request` | Parâmetros inválidos |
| `401 Unauthorized` | Token ausente ou inválido |
| `403 Forbidden` | Sem permissão |
| `404 Not Found` | Recurso não encontrado |
| `500 Internal Server Error` | Erro interno |
| `503 Service Unavailable` | Serviço indisponível |

---

## 💡 Exemplos

### Obter Status de uma Chamada

```bash
curl -X GET \
  http://localhost:8080/api/v1/calls/123e4567-e89b-12d3-a456-426614174000 \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIs...'
```

### Encerrar uma Chamada

```bash
curl -X DELETE \
  http://localhost:8080/api/v1/calls/123e4567-e89b-12d3-a456-426614174000 \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIs...' \
  -H 'Content-Type: application/json' \
  -d '{"reason": "user_hangup"}'
```

### Transferir Chamada para Fila

```bash
curl -X POST \
  http://localhost:8080/api/v1/calls/123e4567-e89b-12d3-a456-426614174000/transfer \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIs...' \
  -H 'Content-Type: application/json' \
  -d '{
    "type": "queue",
    "target": "support-queue",
    "reason": "customer_request"
  }'
```

### Listar Chamadas Ativas

```bash
curl -X GET \
  'http://localhost:8080/api/v1/tenants/987fcdeb-51a2-43d7-8f9e-123456789abc/calls?state=active&limit=10' \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIs...'
```

---

## 🔄 Webhooks (Client-Side)

Para receber notificações de eventos de chamadas, configure webhooks no tenant-manager:

```json
{
  "webhook_url": "https://your-app.com/webhooks/voice",
  "events": [
    "call.started",
    "call.answered",
    "call.ended",
    "call.transferred"
  ],
  "secret": "your-webhook-secret"
}
```

**Exemplo de Evento via Webhook**

```json
{
  "event": "call.answered",
  "timestamp": "2024-01-15T10:30:15.000Z",
  "data": {
    "call_id": "123e4567-e89b-12d3-a456-426614174000",
    "tenant_id": "987fcdeb-51a2-43d7-8f9e-123456789abc",
    "caller_number": "+5511999887766",
    "callee_number": "+5511988776655",
    "answered_at": "2024-01-15T10:30:15.000Z"
  }
}
```

---

## 📚 SDKs

SDKs oficiais em desenvolvimento:

- [ ] JavaScript/TypeScript
- [ ] Python
- [ ] Go
- [ ] Java

---

## 🆘 Suporte

- **Documentação**: https://docs.serphona.com/voice-gateway
- **Issues**: https://github.com/serphona/serphona/issues
- **Email**: support@serphona.com

---

**Última Atualização**: 2025-01-03  
**Versão da API**: 1.0.0
