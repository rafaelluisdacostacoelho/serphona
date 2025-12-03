# Tenant Manager - Telephony Extensions

## 1. Overview

This document describes the extensions needed in the `tenant-manager` service to support telephony configuration for voice-based AI agents. These extensions follow the existing hexagonal architecture pattern already in place.

## 2. New Domain Entities

### 2.1 Trunk (SIP Trunk Configuration)

```go
// internal/domain/telephony/trunk.go
package telephony

import (
	"time"
	"github.com/google/uuid"
)

// Trunk represents a SIP trunk configuration for a tenant
type Trunk struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Provider    string    `json:"provider"` // twilio, bandwidth, custom
	
	// SIP Configuration
	SIPConfig   SIPConfig `json:"sip_config"`
	
	// Capacity and Limits
	MaxConcurrentCalls int    `json:"max_concurrent_calls"`
	CurrentCalls       int    `json:"current_calls"`
	
	// Status
	Status      TrunkStatus `json:"status"` // active, inactive, suspended
	Enabled     bool        `json:"enabled"`
	
	// Metadata
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type SIPConfig struct {
	// Authentication
	Username    string   `json:"username"`
	Password    string   `json:"-"` // Never expose in JSON
	Realm       string   `json:"realm"`
	
	// Connection
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Transport   string   `json:"transport"` // udp, tcp, tls
	
	// Codecs
	Codecs      []string `json:"codecs"` // ulaw, alaw, g729, opus
	
	// Advanced
	Context     string   `json:"context"` // Asterisk context
	RegisterRequired bool `json:"register_required"`
}

type TrunkStatus string

const (
	TrunkStatusActive    TrunkStatus = "active"
	TrunkStatusInactive  TrunkStatus = "inactive"
	TrunkStatusSuspended TrunkStatus = "suspended"
)
```

### 2.2 DID (Direct Inward Dialing / Phone Numbers)

```go
// internal/domain/telephony/did.go
package telephony

import (
	"time"
	"github.com/google/uuid"
)

// DID represents a phone number assigned to a tenant
type DID struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	TrunkID     uuid.UUID  `json:"trunk_id"`
	
	// Number Information
	PhoneNumber string     `json:"phone_number"` // E.164 format: +5511999998888
	CountryCode string     `json:"country_code"` // BR, US, etc
	Type        DIDType    `json:"type"`         // local, toll-free, mobile
	
	// Routing
	RoutingConfig RoutingConfig `json:"routing_config"`
	
	// Status
	Status      DIDStatus  `json:"status"`
	Enabled     bool       `json:"enabled"`
	
	// Metadata
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type DIDType string

const (
	DIDTypeLocal    DIDType = "local"
	DIDTypeTollFree DIDType = "toll-free"
	DIDTypeMobile   DIDType = "mobile"
)

type DIDStatus string

const (
	DIDStatusActive   DIDStatus = "active"
	DIDStatusInactive DIDStatus = "inactive"
)

type RoutingConfig struct {
	// Agent routing
	DefaultAgentID string   `json:"default_agent_id"`
	
	// Business hours routing
	BusinessHours  BusinessHoursRouting `json:"business_hours"`
	AfterHours     AfterHoursRouting    `json:"after_hours"`
	
	// Overflow handling
	MaxQueueTime   int    `json:"max_queue_time"` // seconds
	OverflowAction string `json:"overflow_action"` // voicemail, redirect, hangup
	OverflowTarget string `json:"overflow_target,omitempty"` // phone number or voicemail box
}

type BusinessHoursRouting struct {
	Enabled    bool     `json:"enabled"`
	Schedule   Schedule `json:"schedule"`
	AgentID    string   `json:"agent_id"`
}

type AfterHoursRouting struct {
	Enabled    bool   `json:"enabled"`
	Message    string `json:"message"` // TTS message
	Action     string `json:"action"`  // voicemail, redirect, hangup
}

type Schedule struct {
	Timezone string          `json:"timezone"` // America/Sao_Paulo
	Days     map[string]Hours `json:"days"`    // monday: {start: "09:00", end: "18:00"}
}

type Hours struct {
	Start string `json:"start"` // HH:MM format
	End   string `json:"end"`
}
```

### 2.3 Queue Configuration

```go
// internal/domain/telephony/queue.go
package telephony

import (
	"time"
	"github.com/google/uuid"
)

// Queue represents a call queue for human agents
type Queue struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	
	// Queue Configuration
	Strategy    QueueStrategy `json:"strategy"` // ringall, roundrobin, leastrecent
	Timeout     int           `json:"timeout"`  // seconds to ring each agent
	Retry       int           `json:"retry"`    // seconds between retries
	MaxWait     int           `json:"max_wait"` // max wait time in queue
	
	// Music on Hold
	MusicOnHold string `json:"music_on_hold"` // audio file path or stream URL
	
	// Announcements
	JoinAnnouncement   string `json:"join_announcement,omitempty"`
	PeriodicAnnounce   string `json:"periodic_announce,omitempty"`
	AnnounceFrequency  int    `json:"announce_frequency"` // seconds
	
	// Members (human agents)
	Members     []QueueMember `json:"members"`
	
	// Status
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type QueueStrategy string

const (
	QueueStrategyRingAll      QueueStrategy = "ringall"
	QueueStrategyRoundRobin   QueueStrategy = "roundrobin"
	QueueStrategyLeastRecent  QueueStrategy = "leastrecent"
	QueueStrategyFewestCalls  QueueStrategy = "fewestcalls"
)

type QueueMember struct {
	AgentID    string `json:"agent_id"`    // SIP endpoint or extension
	Penalty    int    `json:"penalty"`     // lower = higher priority
	Paused     bool   `json:"paused"`
	StateInterface string `json:"state_interface,omitempty"`
}
```

### 2.4 Provider Settings (STT/TTS/LLM per tenant)

```go
// internal/domain/telephony/provider_settings.go
package telephony

import (
	"time"
	"github.com/google/uuid"
)

// ProviderSettings stores STT/TTS/LLM provider configurations per tenant
type ProviderSettings struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	
	// Speech-to-Text
	STTProvider STTProviderConfig `json:"stt_provider"`
	
	// Text-to-Speech
	TTSProvider TTSProviderConfig `json:"tts_provider"`
	
	// LLM Provider
	LLMProvider LLMProviderConfig `json:"llm_provider"`
	
	// Metadata
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type STTProviderConfig struct {
	Provider string `json:"provider"` // google, azure, aws, whisper
	Enabled  bool   `json:"enabled"`
	
	// Credentials (encrypted at rest)
	APIKey   string `json:"-"`
	Region   string `json:"region,omitempty"`
	
	// Configuration
	Language string   `json:"language"` // pt-BR, en-US
	Model    string   `json:"model,omitempty"`
	
	// Advanced
	EnableInterimResults bool     `json:"enable_interim_results"`
	ProfanityFilter      bool     `json:"profanity_filter"`
	SampleRate           int      `json:"sample_rate"` // 16000, 8000
}

type TTSProviderConfig struct {
	Provider string `json:"provider"` // google, azure, aws, elevenlabs
	Enabled  bool   `json:"enabled"`
	
	// Credentials
	APIKey   string `json:"-"`
	Region   string `json:"region,omitempty"`
	
	// Configuration
	VoiceID     string  `json:"voice_id"`
	Language    string  `json:"language"`
	SpeechRate  float64 `json:"speech_rate"` // 0.5 to 2.0
	Pitch       float64 `json:"pitch"`       // -20.0 to 20.0
	
	// Advanced
	AudioEncoding string `json:"audio_encoding"` // pcm, mp3, opus
	SampleRate    int    `json:"sample_rate"`
}

type LLMProviderConfig struct {
	Provider string `json:"provider"` // openai, anthropic, azure, custom
	Enabled  bool   `json:"enabled"`
	
	// Credentials
	APIKey      string `json:"-"`
	Endpoint    string `json:"endpoint,omitempty"` // for custom providers
	
	// Configuration
	Model       string  `json:"model"` // gpt-4, claude-3, etc
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
	
	// Advanced
	StreamingEnabled bool `json:"streaming_enabled"`
}
```

## 3. Repository Interfaces

```go
// internal/domain/telephony/repository.go
package telephony

import (
	"context"
	"github.com/google/uuid"
)

// TrunkRepository defines trunk persistence operations
type TrunkRepository interface {
	Create(ctx context.Context, trunk *Trunk) error
	GetByID(ctx context.Context, id uuid.UUID) (*Trunk, error)
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*Trunk, error)
	Update(ctx context.Context, trunk *Trunk) error
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementCallCount(ctx context.Context, trunkID uuid.UUID) error
	DecrementCallCount(ctx context.Context, trunkID uuid.UUID) error
}

// DIDRepository defines DID persistence operations
type DIDRepository interface {
	Create(ctx context.Context, did *DID) error
	GetByID(ctx context.Context, id uuid.UUID) (*DID, error)
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*DID, error)
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*DID, error)
	Update(ctx context.Context, did *DID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// QueueRepository defines queue persistence operations
type QueueRepository interface {
	Create(ctx context.Context, queue *Queue) error
	GetByID(ctx context.Context, id uuid.UUID) (*Queue, error)
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*Queue, error)
	Update(ctx context.Context, queue *Queue) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProviderSettingsRepository defines provider settings operations
type ProviderSettingsRepository interface {
	Create(ctx context.Context, settings *ProviderSettings) error
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*ProviderSettings, error)
	Update(ctx context.Context, settings *ProviderSettings) error
}
```

## 4. Application Layer (Use Cases)

```go
// internal/application/telephony/service.go
package telephony

import (
	"context"
	"github.com/google/uuid"
	"go.uber.org/zap"
	
	"tenant-manager/internal/domain/telephony"
)

// Service implements telephony-related use cases
type Service struct {
	trunkRepo    telephony.TrunkRepository
	didRepo      telephony.DIDRepository
	queueRepo    telephony.QueueRepository
	providerRepo telephony.ProviderSettingsRepository
	logger       *zap.Logger
}

// NewService creates a new telephony service
func NewService(
	trunkRepo telephony.TrunkRepository,
	didRepo telephony.DIDRepository,
	queueRepo telephony.QueueRepository,
	providerRepo telephony.ProviderSettingsRepository,
	logger *zap.Logger,
) *Service {
	return &Service{
		trunkRepo:    trunkRepo,
		didRepo:      didRepo,
		queueRepo:    queueRepo,
		providerRepo: providerRepo,
		logger:       logger,
	}
}

// Trunk operations
func (s *Service) CreateTrunk(ctx context.Context, cmd CreateTrunkCommand) (*TrunkDTO, error) {
	// Validate, create, persist
}

func (s *Service) GetTrunk(ctx context.Context, id uuid.UUID) (*TrunkDTO, error) {
	// Retrieve trunk
}

func (s *Service) ListTrunks(ctx context.Context, tenantID uuid.UUID) ([]*TrunkDTO, error) {
	// List all trunks for tenant
}

// DID operations
func (s *Service) CreateDID(ctx context.Context, cmd CreateDIDCommand) (*DIDDTO, error) {
	// Validate, create, persist
}

func (s *Service) GetDIDByPhoneNumber(ctx context.Context, phoneNumber string) (*DIDDTO, error) {
	// Lookup DID by phone number (for incoming calls)
}

// Provider settings operations
func (s *Service) GetProviderSettings(ctx context.Context, tenantID uuid.UUID) (*ProviderSettingsDTO, error) {
	// Get STT/TTS/LLM settings for tenant
}

func (s *Service) UpdateProviderSettings(ctx context.Context, cmd UpdateProviderSettingsCommand) error {
	// Update provider configurations
}
```

## 5. HTTP API Endpoints

### 5.1 Trunk Management

```
POST   /api/v1/tenants/{tenant_id}/telephony/trunks
GET    /api/v1/tenants/{tenant_id}/telephony/trunks
GET    /api/v1/tenants/{tenant_id}/telephony/trunks/{trunk_id}
PUT    /api/v1/tenants/{tenant_id}/telephony/trunks/{trunk_id}
DELETE /api/v1/tenants/{tenant_id}/telephony/trunks/{trunk_id}
GET    /api/v1/tenants/{tenant_id}/telephony/trunks/{trunk_id}/status
```

### 5.2 DID Management

```
POST   /api/v1/tenants/{tenant_id}/telephony/dids
GET    /api/v1/tenants/{tenant_id}/telephony/dids
GET    /api/v1/tenants/{tenant_id}/telephony/dids/{did_id}
PUT    /api/v1/tenants/{tenant_id}/telephony/dids/{did_id}
DELETE /api/v1/tenants/{tenant_id}/telephony/dids/{did_id}
GET    /api/v1/telephony/dids/lookup/{phone_number}  # For voice-gateway
```

### 5.3 Queue Management

```
POST   /api/v1/tenants/{tenant_id}/telephony/queues
GET    /api/v1/tenants/{tenant_id}/telephony/queues
GET    /api/v1/tenants/{tenant_id}/telephony/queues/{queue_id}
PUT    /api/v1/tenants/{tenant_id}/telephony/queues/{queue_id}
DELETE /api/v1/tenants/{tenant_id}/telephony/queues/{queue_id}
POST   /api/v1/tenants/{tenant_id}/telephony/queues/{queue_id}/members
DELETE /api/v1/tenants/{tenant_id}/telephony/queues/{queue_id}/members/{member_id}
```

### 5.4 Provider Settings

```
GET /api/v1/tenants/{tenant_id}/telephony/provider-settings
PUT /api/v1/tenants/{tenant_id}/telephony/provider-settings
```

### 5.5 Agent Configuration (prompts.yaml)

```
GET    /api/v1/tenants/{tenant_id}/agent-config
PUT    /api/v1/tenants/{tenant_id}/agent-config
POST   /api/v1/tenants/{tenant_id}/agent-config/validate
GET    /api/v1/tenants/{tenant_id}/agent-config/versions
POST   /api/v1/tenants/{tenant_id}/agent-config/versions/{version}/rollback
```

## 6. Database Migrations

```sql
-- migrations/009_telephony_support.sql

-- Trunks table
CREATE TABLE telephony_trunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    
    -- SIP Configuration (JSONB for flexibility)
    sip_config JSONB NOT NULL,
    
    -- Capacity
    max_concurrent_calls INTEGER NOT NULL DEFAULT 10,
    current_calls INTEGER NOT NULL DEFAULT 0,
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    enabled BOOLEAN NOT NULL DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT check_current_calls CHECK (current_calls >= 0),
    CONSTRAINT check_max_calls CHECK (max_concurrent_calls > 0)
);

CREATE INDEX idx_telephony_trunks_tenant ON telephony_trunks(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_telephony_trunks_status ON telephony_trunks(status) WHERE deleted_at IS NULL;

-- DIDs table
CREATE TABLE telephony_dids (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    trunk_id UUID NOT NULL REFERENCES telephony_trunks(id) ON DELETE CASCADE,
    
    -- Number information
    phone_number VARCHAR(20) NOT NULL UNIQUE, -- E.164 format
    country_code VARCHAR(2) NOT NULL,
    type VARCHAR(20) NOT NULL, -- local, toll-free, mobile
    
    -- Routing (JSONB for flexibility)
    routing_config JSONB NOT NULL DEFAULT '{}',
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    enabled BOOLEAN NOT NULL DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX idx_telephony_dids_phone ON telephony_dids(phone_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_telephony_dids_tenant ON telephony_dids(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_telephony_dids_trunk ON telephony_dids(trunk_id) WHERE deleted_at IS NULL;

-- Queues table
CREATE TABLE telephony_queues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    
    -- Queue configuration
    strategy VARCHAR(20) NOT NULL DEFAULT 'roundrobin',
    timeout INTEGER NOT NULL DEFAULT 30,
    retry INTEGER NOT NULL DEFAULT 5,
    max_wait INTEGER NOT NULL DEFAULT 300,
    
    -- Audio
    music_on_hold VARCHAR(255),
    join_announcement VARCHAR(255),
    periodic_announce VARCHAR(255),
    announce_frequency INTEGER DEFAULT 0,
    
    -- Members (JSONB array)
    members JSONB NOT NULL DEFAULT '[]',
    
    -- Status
    enabled BOOLEAN NOT NULL DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_telephony_queues_tenant ON telephony_queues(tenant_id);
CREATE INDEX idx_telephony_queues_enabled ON telephony_queues(enabled);

-- Provider settings table
CREATE TABLE telephony_provider_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- STT Configuration (JSONB with encrypted credentials)
    stt_config JSONB NOT NULL DEFAULT '{}',
    
    -- TTS Configuration
    tts_config JSONB NOT NULL DEFAULT '{}',
    
    -- LLM Configuration
    llm_config JSONB NOT NULL DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_telephony_provider_tenant ON telephony_provider_settings(tenant_id);

-- Agent configurations table (for prompts.yaml storage)
CREATE TABLE agent_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Configuration
    config_yaml TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Validation
    validated BOOLEAN NOT NULL DEFAULT false,
    validation_errors JSONB,
    
    -- Metadata
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_config_tenant ON agent_configurations(tenant_id);
CREATE INDEX idx_agent_config_active ON agent_configurations(tenant_id, is_active) WHERE is_active = true;
CREATE INDEX idx_agent_config_version ON agent_configurations(tenant_id, version);
```

## 7. Handler Implementation Example

```go
// internal/adapter/http/handler/telephony.go
package handler

import (
	"encoding/json"
	"net/http"
	
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	
	"tenant-manager/internal/application/telephony"
)

// TelephonyHandler handles telephony configuration requests
type TelephonyHandler struct {
	service *telephony.Service
	logger  *zap.Logger
}

// NewTelephonyHandler creates a new handler
func NewTelephonyHandler(service *telephony.Service, logger *zap.Logger) *TelephonyHandler {
	return &TelephonyHandler{
		service: service,
		logger:  logger,
	}
}

// CreateTrunk handles POST /api/v1/tenants/{tenant_id}/telephony/trunks
func (h *TelephonyHandler) CreateTrunk(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "tenant_id"))
	if err != nil {
		http.Error(w, "invalid tenant ID", http.StatusBadRequest)
		return
	}
	
	var req CreateTrunkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	
	cmd := telephony.CreateTrunkCommand{
		TenantID:           tenantID,
		Name:               req.Name,
		Provider:           req.Provider,
		SIPConfig:          req.SIPConfig,
		MaxConcurrentCalls: req.MaxConcurrentCalls,
	}
	
	trunk, err := h.service.CreateTrunk(r.Context(), cmd)
	if err != nil {
		h.logger.Error("failed to create trunk", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(trunk)
}

// GetDIDByPhoneNumber handles GET /api/v1/telephony/dids/lookup/{phone_number}
// This endpoint is used by voice-gateway to lookup tenant configuration for incoming calls
func (h *TelephonyHandler) GetDIDByPhoneNumber(w http.ResponseWriter, r *http.Request) {
	phoneNumber := chi.URLParam(r, "phone_number")
	
	did, err := h.service.GetDIDByPhoneNumber(r.Context(), phoneNumber)
	if err != nil {
		http.Error(w, "DID not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(did)
}

// GetProviderSettings handles GET /api/v1/tenants/{tenant_id}/telephony/provider-settings
func (h *TelephonyHandler) GetProviderSettings(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "tenant_id"))
	if err != nil {
		http.Error(w, "invalid tenant ID", http.StatusBadRequest)
		return
	}
	
	settings, err := h.service.GetProviderSettings(r.Context(), tenantID)
	if err != nil {
		http.Error(w, "settings not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}
```

## 8. Integration with voice-gateway

When voice-gateway receives an incoming call:

1. Extract phone number from call
2. Call `GET /api/v1/telephony/dids/lookup/{phone_number}` on tenant-manager
3. Get `tenant_id`, `trunk_id`, and `routing_config`
4. Call `GET /api/v1/tenants/{tenant_id}/telephony/provider-settings` to get STT/TTS/LLM config
5. Call `GET /api/v1/tenants/{tenant_id}/agent-config` to get prompts.yaml
6. Initialize call with appropriate configuration

## 9. Summary

These extensions to tenant-manager provide:
- Complete telephony configuration per tenant
- Trunk and DID management
- Queue configuration for human agent transfers
- Provider settings for STT/TTS/LLM
- Agent configuration via prompts.yaml
- All following the existing hexagonal architecture pattern
