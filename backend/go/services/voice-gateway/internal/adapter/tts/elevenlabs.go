// Package tts provides Text-to-Speech provider implementations.
package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	authclient "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.uber.org/zap"
)

const (
	elevenLabsBaseURL = "https://api.elevenlabs.io/v1"
)

func ensureTenantHeader(ctx context.Context, headers http.Header) http.Header {
	if tenantID, err := middleware.TenantIDFromContext(ctx); err == nil && tenantID != "" {
		return middleware.EnsureTenantHeader(headers, tenantID)
	}

	return headers
}

// ElevenLabsProviderV2 implements TTS using ElevenLabs API v2.
type ElevenLabsProviderV2 struct {
	apiKey     string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewElevenLabsProviderV2 creates a new ElevenLabs TTS provider.
func NewElevenLabsProviderV2(apiKey string, logger *zap.Logger) (*ElevenLabsProviderV2, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("elevenlabs api key is required")
	}

	logger.Info("elevenlabs tts provider v2 initialized")

	return &ElevenLabsProviderV2{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout:   60 * time.Second,
			Transport: authclient.WithDefaultTransport(nil),
		},
		logger: logger,
	}, nil
}

// Synthesize converts text to audio using ElevenLabs.
func (p *ElevenLabsProviderV2) Synthesize(ctx context.Context, text string, config SynthesizeConfig) (io.Reader, error) {
	voiceID := config.VoiceID
	if voiceID == "" {
		voiceID = "21m00Tcm4TlvDq8ikWAM" // Default voice (Rachel)
	}

	// Build request body
	requestBody := ElevenLabsRequest{
		Text:    text,
		ModelID: p.getModelID(config.Model),
		VoiceSettings: &ElevenLabsVoiceSettings{
			Stability:       p.getStability(config.SpeechRate),
			SimilarityBoost: 0.75,
			Style:           0.0,
			UseSpeakerBoost: true,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/text-to-speech/%s", elevenLabsBaseURL, voiceID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", p.apiKey)
	req.Header.Set("Accept", "audio/mpeg")

	// Ensure tenant header is present when context carries tenant id
	req.Header = ensureTenantHeader(ctx, req.Header)

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elevenlabs api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Read entire response
	audioData, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	p.logger.Debug("speech synthesized with elevenlabs",
		zap.String("text", text),
		zap.String("voice_id", voiceID),
		zap.Int("audio_size", len(audioData)),
	)

	return bytes.NewReader(audioData), nil
}

// StreamSynthesize converts text to audio stream using ElevenLabs.
func (p *ElevenLabsProviderV2) StreamSynthesize(ctx context.Context, text string, config SynthesizeConfig) (io.ReadCloser, error) {
	voiceID := config.VoiceID
	if voiceID == "" {
		voiceID = "21m00Tcm4TlvDq8ikWAM" // Default voice (Rachel)
	}

	// Build request body
	requestBody := ElevenLabsRequest{
		Text:    text,
		ModelID: p.getModelID(config.Model),
		VoiceSettings: &ElevenLabsVoiceSettings{
			Stability:       p.getStability(config.SpeechRate),
			SimilarityBoost: 0.75,
			Style:           0.0,
			UseSpeakerBoost: true,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL for streaming
	url := fmt.Sprintf("%s/text-to-speech/%s/stream", elevenLabsBaseURL, voiceID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", p.apiKey)
	req.Header.Set("Accept", "audio/mpeg")

	// Ensure tenant header is present when context carries tenant id
	req.Header = ensureTenantHeader(ctx, req.Header)

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elevenlabs api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	p.logger.Debug("streaming speech synthesis with elevenlabs",
		zap.String("text", text),
		zap.String("voice_id", voiceID),
	)

	return resp.Body, nil
}

// ListVoices lists available voices from ElevenLabs.
func (p *ElevenLabsProviderV2) ListVoices(ctx context.Context) ([]*ElevenLabsVoice, error) {
	url := fmt.Sprintf("%s/voices", elevenLabsBaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("xi-api-key", p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elevenlabs api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var voicesResp ElevenLabsVoicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&voicesResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return voicesResp.Voices, nil
}

// GetVoice retrieves information about a specific voice.
func (p *ElevenLabsProviderV2) GetVoice(ctx context.Context, voiceID string) (*ElevenLabsVoice, error) {
	url := fmt.Sprintf("%s/voices/%s", elevenLabsBaseURL, voiceID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("xi-api-key", p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elevenlabs api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var voice ElevenLabsVoice
	if err := json.NewDecoder(resp.Body).Decode(&voice); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &voice, nil
}

// GetUserInfo retrieves user subscription information.
func (p *ElevenLabsProviderV2) GetUserInfo(ctx context.Context) (*ElevenLabsUserInfo, error) {
	url := fmt.Sprintf("%s/user", elevenLabsBaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("xi-api-key", p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elevenlabs api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var userInfo ElevenLabsUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &userInfo, nil
}

// GetModels lists available models.
func (p *ElevenLabsProviderV2) GetModels(ctx context.Context) ([]*ElevenLabsModel, error) {
	url := fmt.Sprintf("%s/models", elevenLabsBaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("xi-api-key", p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elevenlabs api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var models []*ElevenLabsModel
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return models, nil
}

// Close closes the ElevenLabs provider.
func (p *ElevenLabsProviderV2) Close() error {
	p.httpClient.CloseIdleConnections()
	return nil
}

// Name returns the provider name.
func (p *ElevenLabsProviderV2) Name() string {
	return "elevenlabs-v2"
}

// getModelID returns the appropriate model ID.
func (p *ElevenLabsProviderV2) getModelID(model string) string {
	if model != "" {
		return model
	}
	return "eleven_monolingual_v1" // Default model
}

// getStability converts speech rate to stability parameter.
func (p *ElevenLabsProviderV2) getStability(speechRate float64) float64 {
	// Inverse relationship: faster speech = lower stability
	if speechRate <= 0 {
		speechRate = 1.0
	}
	stability := 1.0 / speechRate
	if stability > 1.0 {
		stability = 1.0
	}
	if stability < 0.0 {
		stability = 0.0
	}
	return stability
}

// ElevenLabsRequest represents the API request body.
type ElevenLabsRequest struct {
	Text          string                   `json:"text"`
	ModelID       string                   `json:"model_id"`
	VoiceSettings *ElevenLabsVoiceSettings `json:"voice_settings,omitempty"`
}

// ElevenLabsVoiceSettings represents voice settings.
type ElevenLabsVoiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
	Style           float64 `json:"style,omitempty"`
	UseSpeakerBoost bool    `json:"use_speaker_boost,omitempty"`
}

// ElevenLabsVoicesResponse represents the voices list response.
type ElevenLabsVoicesResponse struct {
	Voices []*ElevenLabsVoice `json:"voices"`
}

// ElevenLabsVoice represents a voice.
type ElevenLabsVoice struct {
	VoiceID           string                   `json:"voice_id"`
	Name              string                   `json:"name"`
	Category          string                   `json:"category"`
	Description       string                   `json:"description"`
	PreviewURL        string                   `json:"preview_url"`
	AvailableForTiers []string                 `json:"available_for_tiers"`
	Settings          *ElevenLabsVoiceSettings `json:"settings,omitempty"`
	Labels            map[string]string        `json:"labels,omitempty"`
	Samples           []*ElevenLabsVoiceSample `json:"samples,omitempty"`
}

// ElevenLabsVoiceSample represents a voice sample.
type ElevenLabsVoiceSample struct {
	SampleID  string `json:"sample_id"`
	FileName  string `json:"file_name"`
	MimeType  string `json:"mime_type"`
	SizeBytes int    `json:"size_bytes"`
	Hash      string `json:"hash"`
}

// ElevenLabsUserInfo represents user subscription information.
type ElevenLabsUserInfo struct {
	SubscriptionTier     string `json:"subscription_tier"`
	CharacterCount       int    `json:"character_count"`
	CharacterLimit       int    `json:"character_limit"`
	CanExtendCharLimit   bool   `json:"can_extend_character_limit"`
	AllowedToExtendLimit bool   `json:"allowed_to_extend_character_limit"`
}

// ElevenLabsModel represents a TTS model.
type ElevenLabsModel struct {
	ModelID            string   `json:"model_id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	CanBeFinetuned     bool     `json:"can_be_finetuned"`
	CanDoTextToSpeech  bool     `json:"can_do_text_to_speech"`
	CanDoVoiceConvert  bool     `json:"can_do_voice_conversion"`
	Languages          []string `json:"languages"`
	MaxCharactersInput int      `json:"max_characters_request_free_user"`
}
