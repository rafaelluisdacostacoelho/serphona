package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	authclient "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

// Config holds OpenAI embedding settings.
type Config struct {
	APIKey  string
	Model   string
	BaseURL string
	Timeout time.Duration
}

// Client implements embedding.Client for OpenAI-compatible APIs.
type Client struct {
	cfg    Config
	client *http.Client
}

// New creates a new OpenAI embedding client.
func New(cfg Config) (Client, error) {
	if cfg.APIKey == "" {
		return Client{}, errors.New("openai api key required")
	}
	if cfg.Model == "" {
		cfg.Model = "text-embedding-3-small"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 15 * time.Second
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	cfg.BaseURL = base

	return Client{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout, Transport: authclient.WithDefaultTransport(nil)}}, nil
}

// request payload
type embedRequest struct {
	Input interface{} `json:"input"`
	Model string      `json:"model"`
}

type embedData struct {
	Embedding []float32 `json:"embedding"`
}

type embedResponse struct {
	Data []embedData `json:"data"`
}

// Embed calls the OpenAI embeddings endpoint.
func (c Client) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, errors.New("no inputs provided")
	}

	payload := embedRequest{Input: inputs, Model: c.cfg.Model}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	// Adiciona o cabeçalho de tenant usando EnsureTenantHeader
	if tenantID, err := middleware.TenantIDFromContext(ctx); err == nil && tenantID != "" {
		req.Header = middleware.EnsureTenantHeader(req.Header, tenantID)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding request failed: status %d", resp.StatusCode)
	}

	var parsed embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(parsed.Data) == 0 {
		return nil, errors.New("no embeddings returned")
	}

	out := make([][]float32, 0, len(parsed.Data))
	for _, d := range parsed.Data {
		out = append(out, d.Embedding)
	}

	return out, nil
}
