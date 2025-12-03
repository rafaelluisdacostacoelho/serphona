// Package stt provides Speech-to-Text provider implementations.
package stt

import (
	"context"
	"io"
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
	Language        string // Language code (e.g., "pt-BR", "en-US")
	SampleRate      int    // Sample rate in Hz (e.g., 16000)
	Encoding        string // Audio encoding (e.g., "pcm", "opus")
	EnableInterim   bool   // Enable interim (partial) results
	MaxAlternatives int    // Maximum number of alternatives
	ProfanityFilter bool   // Enable profanity filtering
	Model           string // Model to use (provider-specific)
	SingleUtterance bool   // Stop after first utterance
}

// Result represents a transcription result.
type Result struct {
	// Transcript is the recognized text
	Transcript string

	// Confidence is the recognition confidence (0.0 to 1.0)
	Confidence float64

	// IsFinal indicates if this is a final result (not interim)
	IsFinal bool

	// Alternatives contains alternative transcriptions
	Alternatives []Alternative

	// Error contains any error that occurred
	Error error
}

// Alternative represents an alternative transcription.
type Alternative struct {
	Transcript string
	Confidence float64
}

// ProviderType represents supported STT providers.
type ProviderType string

const (
	ProviderGoogle  ProviderType = "google"
	ProviderAzure   ProviderType = "azure"
	ProviderAWS     ProviderType = "aws"
	ProviderWhisper ProviderType = "whisper"
)
