// Package tts provides Text-to-Speech provider implementations.
package tts

import (
	"context"
	"io"
)

// Provider defines the interface for Text-to-Speech providers.
type Provider interface {
	// Synthesize converts text to audio.
	Synthesize(ctx context.Context, text string, config SynthesizeConfig) (io.Reader, error)

	// StreamSynthesize converts text to audio stream.
	StreamSynthesize(ctx context.Context, text string, config SynthesizeConfig) (io.ReadCloser, error)

	// Close closes the provider connection.
	Close() error

	// Name returns the provider name.
	Name() string
}

// SynthesizeConfig contains configuration for speech synthesis.
type SynthesizeConfig struct {
	Language      string  // Language code (e.g., "pt-BR", "en-US")
	VoiceID       string  // Voice ID (provider-specific)
	Gender        string  // Voice gender: "male", "female", "neutral"
	SpeechRate    float64 // Speech rate (0.25 to 4.0, default 1.0)
	Pitch         float64 // Voice pitch (-20.0 to 20.0, default 0.0)
	Volume        float64 // Audio volume (0.0 to 1.0, default 1.0)
	SampleRate    int     // Sample rate in Hz (e.g., 16000, 24000)
	AudioEncoding string  // Audio encoding (e.g., "pcm", "mp3", "opus")
	Model         string  // Model to use (provider-specific)
}

// AudioFormat represents supported audio formats.
type AudioFormat string

const (
	AudioFormatPCM  AudioFormat = "pcm"
	AudioFormatMP3  AudioFormat = "mp3"
	AudioFormatOPUS AudioFormat = "opus"
	AudioFormatWAV  AudioFormat = "wav"
)

// ProviderType represents supported TTS providers.
type ProviderType string

const (
	ProviderGoogle     ProviderType = "google"
	ProviderAzure      ProviderType = "azure"
	ProviderAWS        ProviderType = "aws"
	ProviderElevenLabs ProviderType = "elevenlabs"
)

// Voice represents a TTS voice.
type Voice struct {
	ID       string
	Name     string
	Language string
	Gender   string
	Provider string
}
