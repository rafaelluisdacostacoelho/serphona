package embedding

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Tokenizer is a minimal interface to count tokens for the model; allows pluggable implementations.
type Tokenizer interface {
	CountTokens(text string) (int, error)
}

// OpenAIClient wraps the subset of OpenAI/Azure OpenAI API we need.
type OpenAIClient interface {
	CreateEmbeddings(ctx context.Context, req OpenAIEmbeddingRequest) (OpenAIEmbeddingResponse, error)
}

// OpenAIConfig holds settings for the OpenAI embedding adapter.
type OpenAIConfig struct {
	Model           string
	MaxTokens       int
	Timeout         time.Duration
	RetryMax        int
	RetryBackoffMin time.Duration
	RetryBackoffMax time.Duration
	ExpectedDim     int
}

// OpenAIAdapter implements Client using an OpenAI-like client with retries and dimension checks.
type OpenAIAdapter struct {
	client    OpenAIClient
	tokenizer Tokenizer
	config    OpenAIConfig
}

// NewOpenAIAdapter builds an adapter.
func NewOpenAIAdapter(client OpenAIClient, tokenizer Tokenizer, cfg OpenAIConfig) (*OpenAIAdapter, error) {
	if client == nil {
		return nil, errors.New("client is required")
	}
	if tokenizer == nil {
		return nil, errors.New("tokenizer is required")
	}
	if cfg.Model == "" {
		return nil, errors.New("model is required")
	}
	if cfg.RetryMax <= 0 {
		cfg.RetryMax = 3
	}
	if cfg.RetryBackoffMin <= 0 {
		cfg.RetryBackoffMin = 100 * time.Millisecond
	}
	if cfg.RetryBackoffMax <= 0 {
		cfg.RetryBackoffMax = 2 * time.Second
	}
	return &OpenAIAdapter{client: client, tokenizer: tokenizer, config: cfg}, nil
}

// Embed implements Client. It enforces MaxTokens per input, retries transient errors, and checks embedding dimension when ExpectedDim>0.
func (a *OpenAIAdapter) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	for i, text := range inputs {
		tokens, err := a.tokenizer.CountTokens(text)
		if err != nil {
			return nil, fmt.Errorf("count tokens for input %d: %w", i, err)
		}
		if a.config.MaxTokens > 0 && tokens > a.config.MaxTokens {
			return nil, fmt.Errorf("input %d exceeds max tokens: %d > %d", i, tokens, a.config.MaxTokens)
		}
	}

	req := OpenAIEmbeddingRequest{Model: a.config.Model, Inputs: inputs}

	var resp OpenAIEmbeddingResponse
	var err error
	backoff := a.config.RetryBackoffMin

	ctx, cancel := withTimeout(ctx, a.config.Timeout)
	defer cancel()

	for attempt := 0; attempt < a.config.RetryMax; attempt++ {
		resp, err = a.client.CreateEmbeddings(ctx, req)
		if err == nil {
			break
		}
		if attempt == a.config.RetryMax-1 {
			break
		}
		t := backoff
		backoff *= 2
		if backoff > a.config.RetryBackoffMax {
			backoff = a.config.RetryBackoffMax
		}
		time.Sleep(t)
	}

	if err != nil {
		return nil, err
	}

	embs := make([][]float32, len(resp.Data))
	for i, item := range resp.Data {
		if a.config.ExpectedDim > 0 && len(item.Embedding) != a.config.ExpectedDim {
			return nil, fmt.Errorf("embedding dimension mismatch for item %d: got %d, expected %d", i, len(item.Embedding), a.config.ExpectedDim)
		}
		embs[i] = item.Embedding
	}

	return embs, nil
}

// OpenAIEmbeddingRequest is the payload to the client.
type OpenAIEmbeddingRequest struct {
	Model  string
	Inputs []string
}

// OpenAIEmbeddingResponse mirrors minimal fields from the API.
type OpenAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32
	}
}

// withTimeout mirrors the helper used in pgvector observer to keep behavior consistent.
func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
