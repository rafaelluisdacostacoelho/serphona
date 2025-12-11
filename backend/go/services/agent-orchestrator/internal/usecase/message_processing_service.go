package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/service"
)

// MessageProcessingService defines the interface for message processing
type MessageProcessingService interface {
	// ProcessMessage processes a user message and returns assistant response
	ProcessMessage(ctx context.Context, request *ProcessMessageRequest) (*ProcessMessageResponse, error)
}

// ProcessMessageRequest represents a message processing request
type ProcessMessageRequest struct {
	SessionID uuid.UUID
	Content   string
	UserID    uuid.UUID
}

// ProcessMessageResponse represents the processing result
type ProcessMessageResponse struct {
	MessageID uuid.UUID
	Content   string
	ToolCalls []entity.ToolCall
	Metadata  entity.MessageMetadata
}

// messageProcessingServiceImpl implements MessageProcessingService
type messageProcessingServiceImpl struct {
	sessionService SessionService
	agentService   AgentService
	clientPool     service.LLMClientPool
	toolsClient    service.ToolsClient
}

// NewMessageProcessingService creates a new MessageProcessingService
func NewMessageProcessingService(
	sessionService SessionService,
	agentService AgentService,
	clientPool service.LLMClientPool,
	toolsClient service.ToolsClient,
) MessageProcessingService {
	return &messageProcessingServiceImpl{
		sessionService: sessionService,
		agentService:   agentService,
		clientPool:     clientPool,
		toolsClient:    toolsClient,
	}
}

// ProcessMessage processes a user message and returns assistant response
func (s *messageProcessingServiceImpl) ProcessMessage(
	ctx context.Context,
	request *ProcessMessageRequest,
) (*ProcessMessageResponse, error) {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionService.GetSession(ctx, request.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// 2. Create user message
	userMessage := entity.NewUserMessage(request.SessionID, request.Content)

	// 3. Add user message to session
	if err := s.sessionService.AddMessage(ctx, request.SessionID, userMessage); err != nil {
		return nil, fmt.Errorf("failed to add user message: %w", err)
	}

	// 4. Get agent (use current agent or default)
	var agent *entity.Agent
	if session.Context.CurrentAgent != "" {
		agent, err = s.agentService.GetAgentByName(ctx, session.TenantID, session.Context.CurrentAgent)
		if err != nil {
			return nil, fmt.Errorf("failed to get agent: %w", err)
		}
	} else {
		// Get first active agent for tenant as default
		agents, err := s.agentService.ListAgents(ctx, session.TenantID, true)
		if err != nil || len(agents) == 0 {
			return nil, fmt.Errorf("no active agents found for tenant")
		}
		agent = agents[0]
		session.SetCurrentAgent(agent.Name)
	}

	// 5. Get conversation history
	messages, err := s.sessionService.GetMessages(ctx, request.SessionID, 50, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// 6. Build LLM request
	llmRequest := &service.ChatRequest{
		Messages:    s.convertMessages(messages, agent.SystemPrompt),
		Temperature: agent.Temperature,
		MaxTokens:   agent.MaxTokens,
		Tools:       s.convertTools(agent.Tools),
		ToolChoice:  "auto",
	}

	// 7. Call LLM
	llmResponse, err := s.clientPool.Chat(ctx, agent.Model, llmRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}

	// 8. Process tool calls if present
	if len(llmResponse.ToolCalls) > 0 && s.toolsClient != nil {
		// Create assistant message with tool calls
		assistantMessage := entity.NewAssistantMessage(request.SessionID, llmResponse.Content)
		for _, tc := range llmResponse.ToolCalls {
			assistantMessage.AddToolCall(tc)
		}

		// Add to session
		if err := s.sessionService.AddMessage(ctx, request.SessionID, assistantMessage); err != nil {
			return nil, fmt.Errorf("failed to add assistant message with tool calls: %w", err)
		}

		// Execute tools
		toolResults, err := s.processToolCalls(ctx, session, llmResponse.ToolCalls)
		if err != nil {
			return nil, fmt.Errorf("failed to process tool calls: %w", err)
		}

		// Add tool result messages to session
		for _, toolResult := range toolResults {
			toolMsg := entity.NewToolMessage(request.SessionID, toolResult.ToolCallID, toolResult.Content)
			if err := s.sessionService.AddMessage(ctx, request.SessionID, toolMsg); err != nil {
				return nil, fmt.Errorf("failed to add tool result message: %w", err)
			}
		}

		// Get updated messages for second LLM call
		updatedMessages, err := s.sessionService.GetMessages(ctx, request.SessionID, 50, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated messages: %w", err)
		}

		// Second LLM call with tool results
		secondRequest := &service.ChatRequest{
			Messages:    s.convertMessages(updatedMessages, agent.SystemPrompt),
			Temperature: agent.Temperature,
			MaxTokens:   agent.MaxTokens,
			Tools:       s.convertTools(agent.Tools),
			ToolChoice:  "auto",
		}

		llmResponse, err = s.clientPool.Chat(ctx, agent.Model, secondRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to call LLM with tool results: %w", err)
		}
	}

	// 9. Create final assistant message
	assistantMessage := entity.NewAssistantMessage(request.SessionID, llmResponse.Content)

	// Add tool calls if present (shouldn't happen on second call)
	if len(llmResponse.ToolCalls) > 0 {
		for _, tc := range llmResponse.ToolCalls {
			assistantMessage.AddToolCall(tc)
		}
	}

	// Calculate latency
	latency := time.Since(startTime).Milliseconds()

	// Set metadata
	metadata := entity.MessageMetadata{
		Model:        agent.Model,
		Tokens:       llmResponse.Usage.TotalTokens,
		InputTokens:  llmResponse.Usage.PromptTokens,
		OutputTokens: llmResponse.Usage.CompletionTokens,
		LatencyMS:    latency,
		Agent:        agent.Name,
	}
	assistantMessage.SetMetadata(metadata)
	assistantMessage.CalculateCost()

	// 10. Add final assistant message to session
	if err := s.sessionService.AddMessage(ctx, request.SessionID, assistantMessage); err != nil {
		return nil, fmt.Errorf("failed to add assistant message: %w", err)
	}

	// 11. Build response
	response := &ProcessMessageResponse{
		MessageID: assistantMessage.ID,
		Content:   assistantMessage.Content,
		ToolCalls: assistantMessage.ToolCalls,
		Metadata:  assistantMessage.Metadata,
	}

	return response, nil
}

// Helper methods

func (s *messageProcessingServiceImpl) convertMessages(messages []*entity.Message, systemPrompt string) []entity.Message {
	result := make([]entity.Message, 0, len(messages)+1)

	// Add system message first if present
	if systemPrompt != "" {
		systemMsg := entity.Message{
			ID:      uuid.New(),
			Role:    entity.MessageRoleSystem,
			Content: systemPrompt,
		}
		result = append(result, systemMsg)
	}

	// Add conversation messages
	for _, msg := range messages {
		result = append(result, *msg)
	}

	return result
}

func (s *messageProcessingServiceImpl) convertTools(toolNames []string) []service.Tool {
	// For now, return empty array
	// Tool integration will be done in ToolOrchestrationService
	tools := make([]service.Tool, 0, len(toolNames))

	// TODO: Fetch tool definitions from Tools Gateway
	// For each toolName:
	// - Call Tools Gateway GET /api/v1/tools?name={toolName}
	// - Convert to service.Tool format
	// - Append to tools array

	return tools
}

// ToolResult represents the result of tool execution
type ToolResult struct {
	ToolCallID string
	Content    string
}

// processToolCalls executes tool calls via Tools Gateway
func (s *messageProcessingServiceImpl) processToolCalls(
	ctx context.Context,
	session *entity.Session,
	toolCalls []entity.ToolCall,
) ([]ToolResult, error) {
	results := make([]ToolResult, 0, len(toolCalls))

	for _, toolCall := range toolCalls {
		// Parse tool arguments
		var params map[string]interface{}
		if err := json.Unmarshal(toolCall.Arguments, &params); err != nil {
			// If can't parse, create error result
			results = append(results, ToolResult{
				ToolCallID: toolCall.ID,
				Content:    fmt.Sprintf("Error: failed to parse tool arguments: %v", err),
			})
			continue
		}

		// Execute tool via Tools Gateway
		toolRequest := &service.ToolExecutionRequest{
			ToolName:   toolCall.ToolName,
			Parameters: params,
			TenantID:   session.TenantID,
			UserID:     session.UserID.String(),
		}

		toolResponse, err := s.toolsClient.ExecuteTool(ctx, toolRequest)
		if err != nil {
			// Create error result
			results = append(results, ToolResult{
				ToolCallID: toolCall.ID,
				Content:    fmt.Sprintf("Error executing tool %s: %v", toolCall.ToolName, err),
			})
			continue
		}

		// Convert tool result to string
		var resultContent string
		if toolResponse.Status == "success" {
			// Convert result map to JSON string
			resultJSON, err := json.Marshal(toolResponse.Result)
			if err != nil {
				resultContent = fmt.Sprintf("Error: failed to marshal tool result: %v", err)
			} else {
				resultContent = string(resultJSON)
			}
		} else {
			// Tool execution failed
			resultContent = fmt.Sprintf("Tool execution failed: %s", toolResponse.Error)
		}

		results = append(results, ToolResult{
			ToolCallID: toolCall.ID,
			Content:    resultContent,
		})
	}

	return results, nil
}
