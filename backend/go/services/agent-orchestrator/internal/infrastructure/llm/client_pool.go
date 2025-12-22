package llm

import (
	"context"
	"fmt"
	"sync"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/service"
)

// clientPoolImpl implements service.LLMClientPool
type clientPoolImpl struct {
	clients map[string]service.LLMClient
	mu      sync.RWMutex
}

// NewClientPool creates a new LLM client pool
func NewClientPool() service.LLMClientPool {
	return &clientPoolImpl{
		clients: make(map[string]service.LLMClient),
	}
}

// RegisterClient registers a client for a specific model
func (p *clientPoolImpl) RegisterClient(model string, client service.LLMClient) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients[model] = client
}

// GetClient returns a client for the specified model
func (p *clientPoolImpl) GetClient(model string) (service.LLMClient, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	client, ok := p.clients[model]
	if !ok {
		return nil, fmt.Errorf("no client registered for model: %s", model)
	}

	return client, nil
}

// Chat sends a chat completion request using the appropriate client
func (p *clientPoolImpl) Chat(ctx context.Context, model string, request *service.ChatRequest) (*service.ChatResponse, error) {
	client, err := p.GetClient(model)
	if err != nil {
		return nil, err
	}

	return client.Chat(ctx, request)
}

// Stream sends a streaming chat completion request
func (p *clientPoolImpl) Stream(ctx context.Context, model string, request *service.ChatRequest) (<-chan service.ChatChunk, error) {
	client, err := p.GetClient(model)
	if err != nil {
		return nil, err
	}

	return client.Stream(ctx, request)
}

// SetupDefaultClients sets up clients for common models
func SetupDefaultClients(openaiAPIKey string) service.LLMClientPool {
	pool := NewClientPool().(*clientPoolImpl)

	// Register OpenAI models
	if openaiAPIKey != "" {
		pool.RegisterClient(entity.ModelGPT4Turbo, NewOpenAIClient(openaiAPIKey, entity.ModelGPT4Turbo))
		pool.RegisterClient(entity.ModelGPT4, NewOpenAIClient(openaiAPIKey, entity.ModelGPT4))
		pool.RegisterClient(entity.ModelGPT35Turbo, NewOpenAIClient(openaiAPIKey, entity.ModelGPT35Turbo))
	}

	// TODO: Add Anthropic clients when implemented
	// if anthropicAPIKey != "" {
	//     pool.RegisterClient(entity.ModelClaude3Opus, NewAnthropicClient(anthropicAPIKey, entity.ModelClaude3Opus))
	//     pool.RegisterClient(entity.ModelClaude3Sonnet, NewAnthropicClient(anthropicAPIKey, entity.ModelClaude3Sonnet))
	//     pool.RegisterClient(entity.ModelClaude3Haiku, NewAnthropicClient(anthropicAPIKey, entity.ModelClaude3Haiku))
	// }

	return pool
}
