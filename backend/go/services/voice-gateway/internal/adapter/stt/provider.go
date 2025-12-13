// Package stt provides Speech-to-Text provider implementations.
package stt

import (
	"context"
	"io"
	"time"
)

// Provider defines the interface for Speech-to-Text providers.
type Provider interface {
	// StreamTranscribe transcribes audio stream to text.
	StreamTranscribe(ctx context.Context, audioStream io.Reader, config StreamConfig) (<-chan Result, error)

	// Close closes the provider connection.
	Close() error

	// Name returns the provider name.
	Name() string
}

// StreamConfig contains configuration for streaming transcription.
type StreamConfig struct {
	Language           string   // Language code (e.g., "pt-BR", "en-US")
	SampleRate         int      // Sample rate in Hz (e.g., 16000)
	Encoding           string   // Audio encoding (e.g., "pcm", "opus")
	Channels           int      // Number of audio channels (default: 1)
	EnableInterim      bool     // Enable interim (partial) results
	InterimResults     bool     // Alias for EnableInterim
	MaxAlternatives    int      // Maximum number of alternatives
	ProfanityFilter    bool     // Enable profanity filtering
	Model              string   // Model to use (provider-specific)
	SingleUtterance    bool     // Stop after first utterance
	EnablePunctuation  bool     // Enable automatic punctuation
	UseEnhanced        bool     // Use enhanced models (Google)
	PhraseHints        []string // Phrase hints for better recognition
	PhraseBoost        float32  // Boost for phrase hints
	EnableWordTimeInfo bool     // Enable word-level timing information
}

// Result represents a transcription result.
type Result struct {
	// Transcript is the recognized text
	Transcript string

	// Confidence is the recognition confidence (0.0 to 1.0)
	Confidence float32

	// IsFinal indicates if this is a final result (not interim)
	IsFinal bool

	// Alternatives contains alternative transcriptions
	Alternatives []Alternative

	// Words contains word-level timing and confidence information
	Words []Word

	// Error contains any error that occurred
	Error error
}

// Alternative represents an alternative transcription.
type Alternative struct {
	Transcript string
	Confidence float32
}

// Word represents word-level information.
type Word struct {
	Word       string
	Confidence float32
	StartTime  time.Duration
	EndTime    time.Duration
}

// ProviderType represents supported STT providers.
type ProviderType string

const (
	ProviderGoogle  ProviderType = "google"
	ProviderAzure   ProviderType = "azure"
	ProviderAWS     ProviderType = "aws"
	ProviderWhisper ProviderType = "whisper"
)
