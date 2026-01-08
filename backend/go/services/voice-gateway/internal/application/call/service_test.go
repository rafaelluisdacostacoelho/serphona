package call

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	redigo "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/types"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/agent"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/asterisk"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/redis"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/stt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/tenant"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/adapter/tts"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/domain/call"
)

const (
	defaultSTTProvider = "google"
	defaultTTSProvider = "elevenlabs"
)

type fakePublisher struct {
	started     bool
	answered    bool
	transferred bool
	ended       bool
}

type stubSTTProvider struct{ name string }

func (p *stubSTTProvider) StreamTranscribe(ctx context.Context, audioStream io.Reader, config stt.StreamConfig) (<-chan stt.Result, error) {
	return nil, nil
}
func (p *stubSTTProvider) Close() error { return nil }
func (p *stubSTTProvider) Name() string { return p.name }

type stubTTSProvider struct{ name string }

func (p *stubTTSProvider) Synthesize(ctx context.Context, text string, config tts.SynthesizeConfig) (io.Reader, error) {
	return nil, nil
}
func (p *stubTTSProvider) StreamSynthesize(ctx context.Context, text string, config tts.SynthesizeConfig) (io.ReadCloser, error) {
	return nil, nil
}
func (p *stubTTSProvider) Close() error { return nil }
func (p *stubTTSProvider) Name() string { return p.name }

func (p *fakePublisher) Publish(ctx context.Context, e *types.Event) error { return nil }
func (p *fakePublisher) PublishAsync(ctx context.Context, e *types.Event)  {}
func (p *fakePublisher) Close() error                                      { return nil }
func (p *fakePublisher) HealthCheck(ctx context.Context) error             { return nil }
func (p *fakePublisher) PublishCallStarted(ctx context.Context, c *call.Call) error {
	p.started = true
	return nil
}
func (p *fakePublisher) PublishCallAnswered(ctx context.Context, c *call.Call) error {
	p.answered = true
	return nil
}
func (p *fakePublisher) PublishCallTransferred(ctx context.Context, c *call.Call) error {
	p.transferred = true
	return nil
}
func (p *fakePublisher) PublishCallEnded(ctx context.Context, c *call.Call) error {
	p.ended = true
	return nil
}
func (p *fakePublisher) PublishSTTTranscribed(ctx context.Context, callID, tenantID, conversationID uuid.UUID, text string, confidence float64, isFinal bool, provider string, latency time.Duration) error {
	return nil
}
func (p *fakePublisher) PublishLLMResponded(ctx context.Context, callID, tenantID, conversationID uuid.UUID, agentID, responseText string, latency time.Duration) error {
	return nil
}
func (p *fakePublisher) PublishTTSGenerated(ctx context.Context, callID, tenantID, conversationID uuid.UUID, text, provider, voiceID string, audioSize int, latency time.Duration) error {
	return nil
}
func (p *fakePublisher) PublishError(ctx context.Context, callID, tenantID uuid.UUID, conversationID *uuid.UUID, errorType, errorMessage, component string) error {
	return nil
}

func newTestRepo(t *testing.T) (*redis.CallStateRepository, func()) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redigo.NewClient(&redigo.Options{Addr: mr.Addr()})
	repo := redis.NewCallStateRepository(client, time.Minute)
	return repo, mr.Close
}

func newAsteriskClient(t *testing.T, handler http.HandlerFunc) *asterisk.ARIClientHTTP {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client, err := asterisk.NewARIClientHTTP(asterisk.ARIConfig{URL: srv.URL, Username: "u", Password: "p", AppName: "app"}, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create ARI client: %v", err)
	}
	return client
}

func TestHandleIncomingCallSuccess(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	t.Cleanup(cleanup)
	pub := &fakePublisher{}

	svc := NewService(nil, repo, pub, nil, nil, nil, nil, defaultSTTProvider, defaultTTSProvider, 5, zap.NewNop())

	tenantID := uuid.New()
	callObj, err := svc.HandleIncomingCall(context.Background(), "chan-1", "+100", "+200", tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callObj.State != call.StateRinging {
		t.Fatalf("expected ringing state, got %s", callObj.State)
	}
	if !pub.started {
		t.Fatalf("expected publish call started")
	}

	loaded, err := repo.Get(context.Background(), callObj.ID)
	if err != nil {
		t.Fatalf("expected persisted call: %v", err)
	}
	if loaded.ChannelID != "chan-1" {
		t.Fatalf("unexpected channel id: %s", loaded.ChannelID)
	}
}

func TestAnswerCallUpdatesStateAndPublishes(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	t.Cleanup(cleanup)
	pub := &fakePublisher{}

	ari := newAsteriskClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	svc := NewService(ari, repo, pub, nil, nil, nil, nil, defaultSTTProvider, defaultTTSProvider, 5, zap.NewNop())

	c := call.NewCall(uuid.New(), call.DirectionInbound, "+100", "+200")
	c.ChannelID = "chan-1"
	_ = repo.Save(context.Background(), c)

	if err := svc.AnswerCall(context.Background(), c.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pub.answered {
		t.Fatalf("expected publish call answered")
	}
	stored, _ := repo.Get(context.Background(), c.ID)
	if stored.State != call.StateAnswered {
		t.Fatalf("expected answered state, got %s", stored.State)
	}
}

func TestTransferCallUpdatesStateAndPublishes(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	t.Cleanup(cleanup)
	pub := &fakePublisher{}

	svc := NewService(nil, repo, pub, nil, nil, nil, nil, defaultSTTProvider, defaultTTSProvider, 5, zap.NewNop())

	c := call.NewCall(uuid.New(), call.DirectionInbound, "+100", "+200")
	c.ChannelID = "chan-1"
	_ = repo.Save(context.Background(), c)

	if err := svc.TransferCall(context.Background(), c.ID, "queue", "support", "reason"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pub.transferred {
		t.Fatalf("expected publish call transferred")
	}
	stored, _ := repo.Get(context.Background(), c.ID)
	if stored.State != call.StateTransferred {
		t.Fatalf("expected transferred state, got %s", stored.State)
	}
}

func TestEndCallHangupFailureStillEnds(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	t.Cleanup(cleanup)
	pub := &fakePublisher{}

	ari := newAsteriskClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	svc := NewService(ari, repo, pub, nil, nil, nil, nil, defaultSTTProvider, defaultTTSProvider, 5, zap.NewNop())

	c := call.NewCall(uuid.New(), call.DirectionInbound, "+100", "+200")
	c.ChannelID = "chan-1"
	c.Answer()
	_ = repo.Save(context.Background(), c)

	if err := svc.EndCall(context.Background(), c.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pub.ended {
		t.Fatalf("expected publish call ended")
	}
	stored, _ := repo.Get(context.Background(), c.ID)
	if stored.State != call.StateEnded {
		t.Fatalf("expected ended state, got %s", stored.State)
	}
	if stored.Duration <= 0 {
		t.Fatalf("expected duration to be set")
	}
}

func TestStartConversationSuccess(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	t.Cleanup(cleanup)
	pub := &fakePublisher{}

	tenantID := uuid.New()
	conversationID := uuid.New()

	tenantSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tenants/" + tenantID.String() + "/agent-config":
			_ = json.NewEncoder(w).Encode(tenant.AgentConfig{AgentID: "agent-1", Name: "Agent One", Voice: tenant.VoiceConfig{Provider: "voicep", VoiceID: "v1"}})
		case "/api/v1/tenants/" + tenantID.String() + "/telephony/provider-settings":
			_ = json.NewEncoder(w).Encode(tenant.ProviderSettings{STTProvider: "google", TTSProvider: "elevenlabs"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(tenantSrv.Close)

	agentSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/conversations" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(agent.ConversationResponse{ConversationID: conversationID, AgentID: "agent-1", AgentName: "Agent One", State: "created"})
	}))
	t.Cleanup(agentSrv.Close)

	tenantClient := tenant.NewClient(tenantSrv.URL, "", "", "", "", zap.NewNop())
	agentClient := agent.NewClient(agentSrv.URL, "", "", "", "", zap.NewNop())

	svc := NewService(nil, repo, pub, tenantClient, agentClient, map[string]stt.Provider{}, map[string]tts.Provider{}, defaultSTTProvider, defaultTTSProvider, 5, zap.NewNop())

	c := call.NewCall(tenantID, call.DirectionInbound, "+100", "+200")
	c.ChannelID = "chan-1"
	c.Answer()
	_ = repo.Save(context.Background(), c)

	if err := svc.StartConversation(context.Background(), c.ID, "agent-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, _ := repo.Get(context.Background(), c.ID)
	if stored.State != call.StateActive {
		t.Fatalf("expected active state, got %s", stored.State)
	}
	if stored.ConversationID != conversationID {
		t.Fatalf("unexpected conversation id: %s", stored.ConversationID)
	}
	if stored.AgentID != "agent-1" {
		t.Fatalf("unexpected agent id: %s", stored.AgentID)
	}
	if stored.STTProvider != "google" || stored.TTSProvider != "elevenlabs" {
		t.Fatalf("providers not set: stt=%s tts=%s", stored.STTProvider, stored.TTSProvider)
	}
	if name, ok := stored.Metadata["agent_name"].(string); !ok || name != "Agent One" {
		t.Fatalf("agent metadata missing name: %+v", stored.Metadata)
	}
	if voiceID, ok := stored.Metadata["agent_voice_id"].(string); !ok || voiceID != "v1" {
		t.Fatalf("agent metadata missing voice id: %+v", stored.Metadata)
	}
}

func TestStartConversationUsesDefaultProvidersWhenTenantSettingsEmpty(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	t.Cleanup(cleanup)
	pub := &fakePublisher{}

	tenantID := uuid.New()
	conversationID := uuid.New()

	tenantSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tenants/" + tenantID.String() + "/agent-config":
			_ = json.NewEncoder(w).Encode(tenant.AgentConfig{AgentID: "agent-1", Name: "Agent One", Voice: tenant.VoiceConfig{Provider: "elevenlabs", VoiceID: "v1"}})
		case "/api/v1/tenants/" + tenantID.String() + "/telephony/provider-settings":
			_ = json.NewEncoder(w).Encode(tenant.ProviderSettings{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(tenantSrv.Close)

	agentSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/conversations" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(agent.ConversationResponse{ConversationID: conversationID, AgentID: "agent-1", AgentName: "Agent One", State: "created"})
	}))
	t.Cleanup(agentSrv.Close)

	tenantClient := tenant.NewClient(tenantSrv.URL, "", "", "", "", zap.NewNop())
	agentClient := agent.NewClient(agentSrv.URL, "", "", "", "", zap.NewNop())

	sttProviders := map[string]stt.Provider{defaultSTTProvider: &stubSTTProvider{name: defaultSTTProvider}}
	ttsProviders := map[string]tts.Provider{defaultTTSProvider: &stubTTSProvider{name: defaultTTSProvider}}

	svc := NewService(nil, repo, pub, tenantClient, agentClient, sttProviders, ttsProviders, defaultSTTProvider, defaultTTSProvider, 5, zap.NewNop())

	c := call.NewCall(tenantID, call.DirectionInbound, "+100", "+200")
	c.ChannelID = "chan-1"
	c.Answer()
	_ = repo.Save(context.Background(), c)

	if err := svc.StartConversation(context.Background(), c.ID, "agent-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, _ := repo.Get(context.Background(), c.ID)
	if stored.STTProvider != defaultSTTProvider {
		t.Fatalf("expected default stt provider, got %s", stored.STTProvider)
	}
	if stored.TTSProvider != defaultTTSProvider {
		t.Fatalf("expected default tts provider, got %s", stored.TTSProvider)
	}
}

func TestStartConversationFallsBackWhenTenantProviderUnknown(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	t.Cleanup(cleanup)
	pub := &fakePublisher{}

	tenantID := uuid.New()

	tenantSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tenants/" + tenantID.String() + "/agent-config":
			_ = json.NewEncoder(w).Encode(tenant.AgentConfig{AgentID: "agent-1", Name: "Agent One", Voice: tenant.VoiceConfig{Provider: "voicep", VoiceID: "v1"}})
		case "/api/v1/tenants/" + tenantID.String() + "/telephony/provider-settings":
			_ = json.NewEncoder(w).Encode(tenant.ProviderSettings{STTProvider: "unknown", TTSProvider: "other"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(tenantSrv.Close)

	agentSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/conversations" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(agent.ConversationResponse{ConversationID: uuid.New(), AgentID: "agent-1", AgentName: "Agent One", State: "created"})
	}))
	t.Cleanup(agentSrv.Close)

	tenantClient := tenant.NewClient(tenantSrv.URL, "", "", "", "", zap.NewNop())
	agentClient := agent.NewClient(agentSrv.URL, "", "", "", "", zap.NewNop())

	sttProviders := map[string]stt.Provider{defaultSTTProvider: &stubSTTProvider{name: defaultSTTProvider}}
	ttsProviders := map[string]tts.Provider{defaultTTSProvider: &stubTTSProvider{name: defaultTTSProvider}}

	svc := NewService(nil, repo, pub, tenantClient, agentClient, sttProviders, ttsProviders, defaultSTTProvider, defaultTTSProvider, 5, zap.NewNop())

	c := call.NewCall(tenantID, call.DirectionInbound, "+100", "+200")
	c.ChannelID = "chan-1"
	c.Answer()
	_ = repo.Save(context.Background(), c)

	if err := svc.StartConversation(context.Background(), c.ID, "agent-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, _ := repo.Get(context.Background(), c.ID)
	if stored.STTProvider != defaultSTTProvider {
		t.Fatalf("expected fallback stt provider, got %s", stored.STTProvider)
	}
	if stored.TTSProvider != defaultTTSProvider {
		t.Fatalf("expected fallback tts provider, got %s", stored.TTSProvider)
	}
}

func TestStartConversationFallsBackToAgentVoiceProvider(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	t.Cleanup(cleanup)
	pub := &fakePublisher{}

	tenantID := uuid.New()

	tenantSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tenants/" + tenantID.String() + "/agent-config":
			_ = json.NewEncoder(w).Encode(tenant.AgentConfig{AgentID: "agent-1", Name: "Agent One", Voice: tenant.VoiceConfig{Provider: "voicep", VoiceID: "v1"}})
		case "/api/v1/tenants/" + tenantID.String() + "/telephony/provider-settings":
			_ = json.NewEncoder(w).Encode(tenant.ProviderSettings{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(tenantSrv.Close)

	agentSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/conversations" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(agent.ConversationResponse{ConversationID: uuid.New(), AgentID: "agent-1", AgentName: "Agent One", State: "created"})
	}))
	t.Cleanup(agentSrv.Close)

	tenantClient := tenant.NewClient(tenantSrv.URL, "", "", "", "", zap.NewNop())
	agentClient := agent.NewClient(agentSrv.URL, "", "", "", "", zap.NewNop())

	sttProviders := map[string]stt.Provider{defaultSTTProvider: &stubSTTProvider{name: defaultSTTProvider}}
	ttsProviders := map[string]tts.Provider{"voicep": &stubTTSProvider{name: "voicep"}}

	svc := NewService(nil, repo, pub, tenantClient, agentClient, sttProviders, ttsProviders, defaultSTTProvider, "", 5, zap.NewNop())

	c := call.NewCall(tenantID, call.DirectionInbound, "+100", "+200")
	c.ChannelID = "chan-1"
	c.Answer()
	_ = repo.Save(context.Background(), c)

	if err := svc.StartConversation(context.Background(), c.ID, "agent-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, _ := repo.Get(context.Background(), c.ID)
	if stored.TTSProvider != "voicep" {
		t.Fatalf("expected agent voice provider fallback, got %s", stored.TTSProvider)
	}
}

func TestStartConversationFailsWhenNotActive(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	t.Cleanup(cleanup)
	pub := &fakePublisher{}

	svc := NewService(nil, repo, pub, nil, nil, nil, nil, defaultSTTProvider, defaultTTSProvider, 5, zap.NewNop())

	c := call.NewCall(uuid.New(), call.DirectionInbound, "+100", "+200")
	_ = repo.Save(context.Background(), c)

	if err := svc.StartConversation(context.Background(), c.ID, "agent-1"); err == nil {
		t.Fatalf("expected error for non-active call")
	}
}

func TestStartConversationFailsOnAgentConfigError(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	t.Cleanup(cleanup)
	pub := &fakePublisher{}

	tenantID := uuid.New()

	tenantSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(tenantSrv.Close)

	tenantClient := tenant.NewClient(tenantSrv.URL, "", "", "", "", zap.NewNop())
	svc := NewService(nil, repo, pub, tenantClient, nil, nil, nil, defaultSTTProvider, defaultTTSProvider, 5, zap.NewNop())

	c := call.NewCall(tenantID, call.DirectionInbound, "+100", "+200")
	c.Answer()
	_ = repo.Save(context.Background(), c)

	if err := svc.StartConversation(context.Background(), c.ID, "agent-1"); err == nil {
		t.Fatalf("expected error from agent config fetch")
	}
}

func TestStartConversationFailsOnProviderSettingsError(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	t.Cleanup(cleanup)
	pub := &fakePublisher{}

	tenantID := uuid.New()

	agentSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(agent.ConversationResponse{ConversationID: uuid.New(), AgentID: "agent-1", AgentName: "Agent One", State: "created"})
	}))
	t.Cleanup(agentSrv.Close)

	tenantSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tenants/" + tenantID.String() + "/agent-config":
			_ = json.NewEncoder(w).Encode(tenant.AgentConfig{AgentID: "agent-1", Name: "Agent One", Voice: tenant.VoiceConfig{Provider: "voicep", VoiceID: "v1"}})
		case "/api/v1/tenants/" + tenantID.String() + "/telephony/provider-settings":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(tenantSrv.Close)

	tenantClient := tenant.NewClient(tenantSrv.URL, "", "", "", "", zap.NewNop())
	agentClient := agent.NewClient(agentSrv.URL, "", "", "", "", zap.NewNop())

	svc := NewService(nil, repo, pub, tenantClient, agentClient, nil, nil, defaultSTTProvider, defaultTTSProvider, 5, zap.NewNop())

	c := call.NewCall(tenantID, call.DirectionInbound, "+100", "+200")
	c.Answer()
	_ = repo.Save(context.Background(), c)

	if err := svc.StartConversation(context.Background(), c.ID, "agent-1"); err == nil {
		t.Fatalf("expected error from provider settings fetch")
	}
}

func TestStartConversationFailsOnAgentCreateError(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	t.Cleanup(cleanup)
	pub := &fakePublisher{}

	tenantID := uuid.New()

	agentSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(agentSrv.Close)

	tenantSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tenants/" + tenantID.String() + "/agent-config":
			_ = json.NewEncoder(w).Encode(tenant.AgentConfig{AgentID: "agent-1", Name: "Agent One", Voice: tenant.VoiceConfig{Provider: "voicep", VoiceID: "v1"}})
		case "/api/v1/tenants/" + tenantID.String() + "/telephony/provider-settings":
			_ = json.NewEncoder(w).Encode(tenant.ProviderSettings{STTProvider: "google", TTSProvider: "elevenlabs"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(tenantSrv.Close)

	tenantClient := tenant.NewClient(tenantSrv.URL, "", "", "", "", zap.NewNop())
	agentClient := agent.NewClient(agentSrv.URL, "", "", "", "", zap.NewNop())

	svc := NewService(nil, repo, pub, tenantClient, agentClient, nil, nil, defaultSTTProvider, defaultTTSProvider, 5, zap.NewNop())

	c := call.NewCall(tenantID, call.DirectionInbound, "+100", "+200")
	c.Answer()
	_ = repo.Save(context.Background(), c)

	if err := svc.StartConversation(context.Background(), c.ID, "agent-1"); err == nil {
		t.Fatalf("expected error from conversation create")
	}
}
