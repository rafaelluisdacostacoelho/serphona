# Tenant Manager - Telephony Extensions (en-US)

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
    Timezone string           `json:"timezone"` // America/Sao_Paulo
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
    QueueStrategyRingAll    QueueStrategy = "ringall"
    QueueStrategyRoundRobin QueueStrategy = "roundrobin"
    QueueStrategyLeastRecent QueueStrategy = "leastrecent"
)

type QueueMember struct {
    AgentID   string `json:"agent_id"`
    Priority  int    `json:"priority"`
    Penalty   int    `json:"penalty"`
}
```

### 2.4 Routing Profiles

```go
// internal/domain/telephony/routing_profile.go
package telephony

import (
    "time"

    "github.com/google/uuid"
)

// RoutingProfile maps intents or tags to agents and tools
type RoutingProfile struct {
    ID          uuid.UUID   `json:"id"`
    TenantID    uuid.UUID   `json:"tenant_id"`
    Name        string      `json:"name"`
    Description string      `json:"description"`

    // Matching
    Tags        []string    `json:"tags"`        // e.g., billing, support
    Keywords    []string    `json:"keywords"`    // regex or keyword list
    Language    string      `json:"language"`    // pt-BR, en-US

    // Routing
    DefaultAgentID string   `json:"default_agent_id"`
    EscalationAgentID string `json:"escalation_agent_id"`

    // Tools (MCP/REST)
    Tools       []ToolBinding `json:"tools"`

    // Metadata
    Enabled     bool        `json:"enabled"`
    CreatedAt   time.Time   `json:"created_at"`
    UpdatedAt   time.Time   `json:"updated_at"`
}

type ToolBinding struct {
    Name        string `json:"name"`        // logical tool name
    Type        string `json:"type"`        // mcp|rest|grpc
    Resource    string `json:"resource"`    // e.g., billing.get_invoice
    Timeout     string `json:"timeout"`     // e.g., 5s
}
```

## 3. Application Layer Changes

### 3.1 Ports (Use Cases)
Add new use cases under `internal/app/telephony`:

- `CreateTrunk`, `UpdateTrunk`, `ListTrunks`, `GetTrunk`, `DeleteTrunk`
- `CreateDID`, `UpdateDID`, `ListDIDs`, `GetDID`, `DeleteDID`
- `CreateQueue`, `UpdateQueue`, `ListQueues`, `GetQueue`, `DeleteQueue`
- `CreateRoutingProfile`, `UpdateRoutingProfile`, `ListRoutingProfiles`, `GetRoutingProfile`, `DeleteRoutingProfile`

Each use case should enforce tenant-level RBAC and validate uniqueness constraints (e.g., phone number per tenant).

### 3.2 Services
Implement services in `internal/app/telephony/service.go` that orchestrate validation, repository access, and domain rules. Examples:

```go
// CreateTrunkService handles creation with validation
func (s *Service) CreateTrunk(ctx context.Context, req CreateTrunkRequest) (*telephony.Trunk, error) {
    if err := s.validator.ValidateTrunk(req); err != nil {
        return nil, err
    }
    trunk := req.ToDomain()
    if err := s.repo.CreateTrunk(ctx, trunk); err != nil {
        return nil, err
    }
    return trunk, nil
}
```

### 3.3 Validation Rules
- Unique phone number per tenant (DID).
- Max concurrent calls per trunk cannot exceed plan limit.
- Queue members must belong to the same tenant.
- Routing profile must reference existing agents and tools for the tenant.

## 4. Infrastructure Layer Changes

### 4.1 Repositories
Add repository interfaces and implementations (PostgreSQL) under `internal/infra/repo/telephony`:
- `TrunkRepository`
- `DIDRepository`
- `QueueRepository`
- `RoutingProfileRepository`

Tables should include `tenant_id` and enforce RLS policies. Example DDL (simplified):

```sql
CREATE TABLE telephony_trunks (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    sip_config JSONB NOT NULL,
    max_concurrent_calls INT NOT NULL,
    current_calls INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ
);
```

### 4.2 REST/gRPC Handlers
Expose CRUD endpoints under `/api/telephony/...` respecting tenant auth middleware:
- `POST /trunks`, `GET /trunks`, `GET /trunks/{id}`, `PUT /trunks/{id}`, `DELETE /trunks/{id}`
- Similarly for DIDs, Queues, Routing Profiles.

### 4.3 gRPC Proto Sketch

```proto
message Trunk {
  string id = 1;
  string tenant_id = 2;
  string name = 3;
  string provider = 4;
  SIPConfig sip_config = 5;
  int32 max_concurrent_calls = 6;
  int32 current_calls = 7;
  string status = 8;
  bool enabled = 9;
}

message SIPConfig {
  string username = 1;
  string realm = 2;
  string host = 3;
  int32 port = 4;
  string transport = 5;
  repeated string codecs = 6;
  bool register_required = 7;
}

service TelephonyService {
  rpc CreateTrunk(CreateTrunkRequest) returns (TrunkResponse);
  rpc ListTrunks(ListTrunksRequest) returns (ListTrunksResponse);
  rpc CreateDID(CreateDIDRequest) returns (DIDResponse);
}
```

## 5. Integration with Voice Stack
- Tenant-manager owns telephony config (trunks, DIDs, queues, routing profiles).
- Voice stack (Kamailio/rtpengine/Asterisk) consumes configs via an adapter/service that pulls from tenant-manager (e.g., gRPC or config export to Redis/Postgres views).
- Changes in tenant-manager should emit events (Kafka) to notify voice components and audit.

## 6. Observability & Audit
- Trace CRUD operations with tenant_id, actor, resource_id.
- Metrics: counts per tenant (trunks, DIDs, queues), provision/update latency, error rates.
- Audit log for telephony changes (who, when, what changed); consider immutable log or append-only table.

## 7. Security & Compliance
- Enforce RLS on all telephony tables by tenant_id.
- Encrypt secrets (SIP passwords) at rest; never expose in APIs; use Vault/KMS.
- Validate phone numbers (E.164), SIP hosts, and codec lists against allowlists.
- Rate-limit admin APIs; require elevated scopes for telephony changes.

## 8. Rollout Plan
- Phase 1: CRUD APIs + repos + RLS + events.
- Phase 2: Adapter for voice stack (read-through cache) + reconciliation loop.
- Phase 3: UI in console for telephony settings; audit dashboards.
- Phase 4: Autoscaling policies for trunks/queues; alarms on capacity.
