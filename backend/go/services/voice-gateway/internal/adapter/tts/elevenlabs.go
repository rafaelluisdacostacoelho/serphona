// Package tts provides Text-to-Speech provider implementations.
package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// ElevenLabsProvider implements TTS using ElevenLabs API.
type ElevenLabsProvider struct {
	apiKey string
	client *http.Client
	logger *zap.Logger
}

// NewElevenLabsProvider creates a new ElevenLabs TTS provider.
func NewElevenLabsProvider(apiKey string, logger *zap.Logger) (*ElevenLabsProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("elevenlabs api key is required")
	}

	logger.Info("elevenlabs tts provider initialized")

	return &ElevenLabsProvider{
		apiKey: apiKey,
		client: &http.Client{},
		logger: logger,
	}, nil
}

// Synthesize converts text to audio using ElevenLabs.
func (p *ElevenLabsProvider) Synthesize(ctx context.Context, text string, config SynthesizeConfig) (io.Reader, error) {
	// TODO: Implement ElevenLabs TTS API call
	// POST https://api.elevenlabs.io/v1/text-to-speech/{voice_id}

	p.logger.Debug("synthesizing speech with elevenlabs",
		zap.String("text", text),
		zap.String("voice_id", config.VoiceID),
	)

	// Placeholder request body
	requestBody := map[string]interface{}{
		"text":     text,
		"model_id": "eleven_monolingual_v1",
		"voice_settings": map[string]interface{}{
			"stability":        0.5,
			"similarity_boost": 0.75,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// TODO: Make actual HTTP request to ElevenLabs API
	_ = jsonData

	// Placeholder
	return bytes.NewReader([]byte{}), nil
}

// StreamSynthesize converts text to audio stream using ElevenLabs.
func (p *ElevenLabsProvider) StreamSynthesize(ctx context.Context, text string, config SynthesizeConfig) (io.ReadCloser, error) {
	// TODO: Implement ElevenLabs streaming TTS
	// POST https://api.elevenlabs.io/v1/text-to-speech/{voice_id}/stream

	p.logger.Debug("streaming speech synthesis with elevenlabs",
		zap.String("text", text),
		zap.String("voice_id", config.VoiceID),
	)

	// ElevenLabs supports streaming
	reader, err := p.Synthesize(ctx, text, config)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(reader), nil
}

// Close closes the ElevenLabs provider.
func (p *ElevenLabsProvider) Close() error {
	p.client.CloseIdleConnections()
	return nil
}

// Name returns the provider name.
func (p *ElevenLabsProvider) Name() string {
	return "elevenlabs"
}
