# Voice Gateway Design - Serphona Platform

## 1. Overview

The **voice-gateway** service is responsible for bridging telephony (Asterisk) with the AI agent orchestration layer. It handles real-time voice interactions, streaming audio to/from STT/TTS providers, and coordinating with agent-orchestrator for LLM-based conversations.

## 2. Responsibilities

### What voice-gateway DOES:
- **Asterisk Integration**: Manage call lifecycle via ARI (Asterisk REST Interface)
- **Audio Streaming**: Handle RTP audio streams from/to Asterisk
- **STT Integration**: Stream audio to Speech-to-Text providers (Google, Azure, AWS, OpenAI Whisper)
- **TTS Integration**: Convert text responses to audio and play back to caller
- **Call State Management**: Track active calls, conversation context, and tenant information
- **Audio Buffering**: Manage audio buffers for streaming and playback
- **Transfer Orchestration**: Execute call transfers to queues or external numbers
- **DTMF Handling**: Process dual-tone multi-frequency signals for IVR-like interactions

### What voice-gateway DOES NOT DO:
- **LLM Processing**: Delegates to agent-orchestrator
- **Tool Execution**: Delegates to tools-gateway
- **Tenant Management**: Queries tenant-manager for configuration
- **Long-term Storage**: Uses platform-events to publish events for analytics
- **Authentication**: Uses platform-auth for JWT validation on management APIs
- **Business Logic**: Pure orchestration, no business rules

## 3. Architecture Principles

- **Hexagonal Architecture**: Clear separation between domain, application, and adapters
- **Event-Driven**: Publishes events via platform-events for analytics
- **Multi-tenant**: All operations scoped by tenant_id
- **Observable**: Integrated with platform-observability
- **Resilient**: Graceful degradation, fallbacks, timeout handling

## 4. Technology Stack

- **Language**: Go 1.23+
- **Asterisk Integration**: ARI (primary) + AMI (fallback)
- **Audio Processing**: Raw PCM, WAV, opus codec support
- **Streaming**: WebSocket for bidirectional audio streaming
- **State Management**: Redis for call state (with in-memory cache)
- **Events**: Kafka via platform-events
- **Metrics**: Prometheus via platform-observability

## 5. Service Structure

```
backend/go/services/voice-gateway/
├── cmd/
│   └── server/
│       └── main.go                    # Service entry point
├── internal/
│   ├── adapter/
│   │   ├── asterisk/
│   │   │   ├── ari_client.go         # ARI HTTP/WebSocket client
│   │   │   ├── call_manager.go       # Call lifecycle management
│   │   │   ├── audio_bridge.go       # Audio streaming bridge
│   │   │   └── dtmf_handler.go       # DTMF event handling
│   │   ├── stt/
│   │   │   ├── provider.go           # STT provider interface
│   │   │   ├── google.go             # Google Speech-to-Text
│   │   │   ├── azure.go              # Azure Speech Services
│   │   │   ├── aws.go                # AWS Transcribe
│   │   │   └── whisper.go            # OpenAI Whisper
│   │   ├── tts/
│   │   │   ├── provider.go           # TTS provider interface
│   │   │   ├── google.go             # Google Text-to-Speech
│   │   │   ├── azure.go              # Azure Speech Services
│   │   │   ├── aws.go                # AWS Polly
│   │   │   └── elevenlabs.go         # ElevenLabs
│   │   ├── agent/
│   │   │   └── orchestrator_client.go # HTTP/gRPC client to agent-orchestrator
│   │   ├── tenant/
│   │   │   └── manager_client.go     # Client to tenant-manager
│   │   ├── http/
│   │   │   ├── handler/
│   │   │   │   ├── health.go         # Health checks
│   │   │   │   ├── call.go           # Call management API
│   │   │   │   └── webhook.go        # Asterisk webhooks
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go           # JWT authentication
│   │   │   │   └── tenant.go         # Tenant context
│   │   │   └── router/
│   │   │       └── router.go         # HTTP routing
│   │   ├── redis/
│   │   │   └── call_state.go         # Call state persistence
│   │   └── events/
│   │       └── publisher.go          # Event publishing
│   ├── application/
│   │   ├── call/
│   │   │   ├── service.go            # Call orchestration service
│   │   │   ├── commands.go           # Call commands
│   │   │   └── queries.go            # Call queries
│   │   ├── conversation/
│   │   │   ├── manager.go            # Conversation state management
│   │   │   └── context.go            # Conversation context
│   │   └── audio/
│   │       ├── processor.go          # Audio processing pipeline
│   │       └── buffer.go             # Audio buffer management
│   ├── domain/
│   │   ├── call/
│   │   │   ├── call.go               # Call aggregate
│   │   │   ├── state.go              # Call states
│   │   │   └── repository.go         # Call repository interface
│   │   ├── conversation/
│   │   │   ├── conversation.go       # Conversation aggregate
│   │   │   └── turn.go               # Conversation turn
│   │   └── audio/
│   │       ├── stream.go             # Audio stream
│   │       └── codec.go              # Audio codec
│   └── config/
│       └── config.go                 # Configuration
├── pkg/
│   ├── audio/
│   │   ├── pcm.go                    # PCM audio utilities
│   │   ├── wav.go                    # WAV file handling
│   │   └── resampler.go              # Audio resampling
│   └── errors/
│       └── errors.go                 # Custom errors
├── migrations/
│   └── 001_initial.sql               # Database migrations (if needed)
├── .env.example
├── Dockerfile
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 6. Call Flow

### Incoming Call Flow:
```
1. Asterisk receives call → triggers ARI webhook to voice-gateway
2. voice-gateway:
   - Extracts tenant_id from DID/trunk configuration
   - Creates call entity with call_id, channel_id
   - Queries tenant-manager for tenant settings (STT/TTS/LLM providers)
   - Initializes conversation with agent-orchestrator
   - Answers the call
3. Audio streaming begins:
   - Asterisk → voice-gateway → STT provider
   - STT provider → text → agent-orchestrator
   - agent-orchestrator → LLM → response text
   - response text → TTS provider → audio
   - audio → voice-gateway → Asterisk → caller
4. Conversation continues in loop until:
   - Caller hangs up
   - Agent triggers transfer
   - Error/timeout occurs
5. voice-gateway:
   - Publishes call events via platform-events
   - Cleans up resources
   - Updates call state to "ended"
```

### Outbound Call Flow:
```
1. External system triggers outbound call via voice-gateway API
2. voice-gateway:
   - Validates tenant_id and permissions
   - Requests Asterisk to originate call via ARI
   - Waits for call answer
3. Once answered, follows same flow as incoming call
```

## 7. State Management

### Call State (Redis):
```json
{
  "call_id": "uuid",
  "tenant_id": "uuid",
  "channel_id": "SIP/trunk-00000001",
  "conversation_id": "uuid",
  "state": "active|ringing|answered|transferred|ended",
  "direction": "inbound|outbound",
  "caller_number": "+5511999998888",
  "callee_number": "+55112345",
  "answered_at": "2024-01-01T10:00:00Z",
  "ended_at": null,
  "agent_id": "tech-support-agent",
  "stt_provider": "google",
  "tts_provider": "elevenlabs",
  "audio_buffer": "reference_to_redis_stream",
  "metadata": {
    "trunk_id": "uuid",
    "did_id": "uuid"
  }
}
```

## 8. Integration Points

### With tenant-manager:
- GET /api/v1/tenants/{id}/telephony/settings
- GET /api/v1/tenants/{id}/telephony/trunks/{trunk_id}
- GET /api/v1/tenants/{id}/telephony/dids/{did}

### With agent-orchestrator:
- POST /api/v1/conversations (create conversation)
- POST /api/v1/conversations/{id}/turns (send user message)
- GET /api/v1/conversations/{id}/agent (get agent for conversation)
- POST /api/v1/conversations/{id}/transfer (request transfer)

### With platform-events:
- Publish: call.started, call.answered, call.ended
- Publish: stt.transcribed, llm.responded, tts.generated
- Publish: call.transferred, call.escalated
- Publish: error.stt_failed, error.llm_timeout, error.tts_failed

## 9. Error Handling Strategy

### STT Failures:
- Timeout: 5 seconds of silence → prompt "I didn't catch that"
- Error: Fallback to DTMF menu or transfer to human
- Retries: 3 attempts before escalation

### LLM Failures:
- Timeout: 10 seconds → fallback response
- Error: Use cached response or transfer
- Rate limit: Queue request or defer to human

### TTS Failures:
- Timeout: 5 seconds → use backup provider
- Error: Play pre-recorded message
- Quality issue: Log and continue

### Network Failures:
- Asterisk disconnect: Attempt reconnect (3x)
- STT/TTS disconnect: Switch to backup provider
- Redis disconnect: Use in-memory cache (5 min TTL)

## 10. Performance Requirements

- **Call Handling**: Support 1000+ concurrent calls per instance
- **STT Latency**: < 500ms for partial transcriptions
- **LLM Latency**: < 2 seconds for response generation
- **TTS Latency**: < 1 second for audio generation
- **End-to-End Latency**: < 3 seconds from speech to response
- **Audio Quality**: 16kHz, 16-bit PCM minimum

## 11. Security Considerations

- **Call Authentication**: Verify trunk credentials via Asterisk
- **API Authentication**: JWT tokens from platform-auth
- **Data Privacy**: Encrypt audio in transit (TLS)
- **PCI Compliance**: Mask credit card data in transcripts
- **GDPR**: Support call recording opt-out
- **Audit**: Log all call events with tenant_id

## 12. Deployment

- **Scaling**: Horizontal scaling via load balancer
- **High Availability**: Multiple instances with shared Redis
- **Resource Allocation**: 2 CPU, 4GB RAM per 100 concurrent calls
- **Network**: Low-latency connection to Asterisk (< 10ms)
