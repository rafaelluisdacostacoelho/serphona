package llm

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/service"
)

// openAIClient implements service.LLMClient for OpenAI
type openAIClient struct {
	client *openai.Client
	model  string
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey, model string) service.LLMClient {
	client := openai.NewClient(apiKey)
	return &openAIClient{
		client: client,
		model:  model,
	}
}

// Chat sends a chat completion request
func (c *openAIClient) Chat(ctx context.Context, request *service.ChatRequest) (*service.ChatResponse, error) {
	// Convert messages
	messages := c.convertMessages(request.Messages)

	// Build request
	req := openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: float32(request.Temperature),
		MaxTokens:   request.MaxTokens,
	}

	// Add tools if present
	if len(request.Tools) > 0 {
		req.Tools = c.convertTools(request.Tools)
		if request.ToolChoice != "" {
			req.ToolChoice = request.ToolChoice
		}
	}

	// Send request
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	// Check if we have choices
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from OpenAI")
	}

	choice := resp.Choices[0]

	// Build response
	response := &service.ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
		Usage: service.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// Convert tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = c.convertToolCallsFromOpenAI(choice.Message.ToolCalls)
	}

	return response, nil
}

// Stream sends a streaming chat completion request
func (c *openAIClient) Stream(ctx context.Context, request *service.ChatRequest) (<-chan service.ChatChunk, error) {
	// Convert messages
	messages := c.convertMessages(request.Messages)

	// Build request
	req := openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: float32(request.Temperature),
		MaxTokens:   request.MaxTokens,
		Stream:      true,
	}

	// Add tools if present
	if len(request.Tools) > 0 {
		req.Tools = c.convertTools(request.Tools)
		if request.ToolChoice != "" {
			req.ToolChoice = request.ToolChoice
		}
	}

	// Create stream
	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	// Create output channel
	chunks := make(chan service.ChatChunk, 10)

	// Process stream in goroutine
	go func() {
		defer close(chunks)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				if err.Error() != "EOF" {
					chunks <- service.ChatChunk{
						Error: fmt.Errorf("stream error: %w", err),
						Done:  true,
					}
				} else {
					chunks <- service.ChatChunk{
						Done: true,
					}
				}
				return
			}

			if len(response.Choices) == 0 {
				continue
			}

			delta := response.Choices[0].Delta

			chunk := service.ChatChunk{
				Content:      delta.Content,
				FinishReason: string(response.Choices[0].FinishReason),
			}

			// Handle tool calls in delta
			if len(delta.ToolCalls) > 0 {
				// Note: Streaming tool calls require accumulation
				// This is a simplified version
				for _, tc := range delta.ToolCalls {
					if tc.Function.Name != "" || tc.Function.Arguments != "" {
						chunk.ToolCall = &entity.ToolCall{
							ID:        tc.ID,
							ToolName:  tc.Function.Name,
							Arguments: []byte(tc.Function.Arguments),
							Status:    entity.ToolCallStatusPending,
						}
					}
				}
			}

			chunks <- chunk
		}
	}()

	return chunks, nil
}

// CountTokens estimates token count for messages
func (c *openAIClient) CountTokens(messages []entity.Message) int {
	// Simple estimation: ~4 characters per token
	// For production, use tiktoken library
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
	}
	return totalChars / 4
}

// GetModel returns the model identifier
func (c *openAIClient) GetModel() string {
	return c.model
}

// Helper methods

func (c *openAIClient) convertMessages(messages []entity.Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(messages))

	for _, msg := range messages {
		oaiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// Add tool calls if present
		if len(msg.ToolCalls) > 0 {
			oaiMsg.ToolCalls = c.convertToolCallsToOpenAI(msg.ToolCalls)
		}

		result = append(result, oaiMsg)
	}

	return result
}

func (c *openAIClient) convertTools(tools []service.Tool) []openai.Tool {
	result := make([]openai.Tool, 0, len(tools))

	for _, tool := range tools {
		oaiTool := openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
		result = append(result, oaiTool)
	}

	return result
}

func (c *openAIClient) convertToolCallsToOpenAI(toolCalls []entity.ToolCall) []openai.ToolCall {
	result := make([]openai.ToolCall, 0, len(toolCalls))

	for _, tc := range toolCalls {
		oaiTC := openai.ToolCall{
			ID:   tc.ID,
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionCall{
				Name:      tc.ToolName,
				Arguments: string(tc.Arguments),
			},
		}
		result = append(result, oaiTC)
	}

	return result
}

func (c *openAIClient) convertToolCallsFromOpenAI(toolCalls []openai.ToolCall) []entity.ToolCall {
	result := make([]entity.ToolCall, 0, len(toolCalls))

	for _, tc := range toolCalls {
		entityTC := entity.ToolCall{
			ID:        tc.ID,
			ToolName:  tc.Function.Name,
			Arguments: []byte(tc.Function.Arguments),
			Status:    entity.ToolCallStatusPending,
		}
		result = append(result, entityTC)
	}

	return result
}
