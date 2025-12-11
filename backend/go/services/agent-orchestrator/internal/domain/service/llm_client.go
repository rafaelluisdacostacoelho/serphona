package service

import (
	"context"

	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
)

// LLMClient defines the interface for LLM providers
type LLMClient interface {
	// Chat sends a chat completion request
	Chat(ctx context.Context, request *ChatRequest) (*ChatResponse, error)

	// Stream sends a streaming chat completion request
	Stream(ctx context.Context, request *ChatRequest) (<-chan ChatChunk, error)

	// CountTokens estimates token count for messages
	CountTokens(messages []entity.Message) int

	// GetModel returns the model identifier
	GetModel() string
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Messages    []entity.Message
	Temperature float64
	MaxTokens   int
	Tools       []Tool
	ToolChoice  string // auto, none, or specific tool name
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Content      string
	ToolCalls    []entity.ToolCall
	FinishReason string
	Usage        TokenUsage
}

// ChatChunk represents a streaming response chunk
type ChatChunk struct {
	Content      string
	ToolCall     *entity.ToolCall
	FinishReason string
	Done         bool
	Error        error
}

// Tool represents a function/tool definition
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{} // JSON Schema
}

// TokenUsage represents token consumption
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// LLMClientPool manages multiple LLM clients
type LLMClientPool interface {
	// GetClient returns a client for the specified model
	GetClient(model string) (LLMClient, error)

	// Chat sends a chat completion request using the appropriate client
	Chat(ctx context.Context, model string, request *ChatRequest) (*ChatResponse, error)

	// Stream sends a streaming chat completion request
	Stream(ctx context.Context, model string, request *ChatRequest) (<-chan ChatChunk, error)
}
