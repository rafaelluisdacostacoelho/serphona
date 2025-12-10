package entity

import (
	"time"

	"github.com/google/uuid"
)

// Agent represents an LLM agent configuration
type Agent struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Name         string    `json:"name"` // Unique identifier
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description"`
	SystemPrompt string    `json:"system_prompt"`
	Model        string    `json:"model"` // gpt-4, claude-3, etc
	Temperature  float64   `json:"temperature"`
	MaxTokens    int       `json:"max_tokens"`
	Tools        []string  `json:"tools"` // tool names from tools-gateway
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Agent model constants
const (
	ModelGPT4Turbo     = "gpt-4-turbo"
	ModelGPT4          = "gpt-4"
	ModelGPT35Turbo    = "gpt-3.5-turbo"
	ModelClaude3Opus   = "claude-3-opus-20240229"
	ModelClaude3Sonnet = "claude-3-sonnet-20240229"
	ModelClaude3Haiku  = "claude-3-haiku-20240307"
)

// NewAgent creates a new agent
func NewAgent(tenantID uuid.UUID, name, displayName, systemPrompt, model string) *Agent {
	now := time.Now()
	return &Agent{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         name,
		DisplayName:  displayName,
		SystemPrompt: systemPrompt,
		Model:        model,
		Temperature:  0.7,  // Default temperature
		MaxTokens:    2000, // Default max tokens
		Tools:        []string{},
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// AddTool adds a tool to the agent
func (a *Agent) AddTool(toolName string) {
	a.Tools = append(a.Tools, toolName)
	a.UpdatedAt = time.Now()
}

// RemoveTool removes a tool from the agent
func (a *Agent) RemoveTool(toolName string) {
	tools := []string{}
	for _, t := range a.Tools {
		if t != toolName {
			tools = append(tools, t)
		}
	}
	a.Tools = tools
	a.UpdatedAt = time.Now()
}

// HasTool checks if the agent has a specific tool
func (a *Agent) HasTool(toolName string) bool {
	for _, t := range a.Tools {
		if t == toolName {
			return true
		}
	}
	return false
}

// HasTools checks if the agent has any tools
func (a *Agent) HasTools() bool {
	return len(a.Tools) > 0
}

// SetSystemPrompt updates the system prompt
func (a *Agent) SetSystemPrompt(prompt string) {
	a.SystemPrompt = prompt
	a.UpdatedAt = time.Now()
}

// SetModel updates the model
func (a *Agent) SetModel(model string) {
	a.Model = model
	a.UpdatedAt = time.Now()
}

// SetTemperature updates the temperature
func (a *Agent) SetTemperature(temp float64) {
	if temp < 0 {
		temp = 0
	}
	if temp > 2 {
		temp = 2
	}
	a.Temperature = temp
	a.UpdatedAt = time.Now()
}

// SetMaxTokens updates the max tokens
func (a *Agent) SetMaxTokens(tokens int) {
	if tokens < 1 {
		tokens = 1
	}
	a.MaxTokens = tokens
	a.UpdatedAt = time.Now()
}

// Activate activates the agent
func (a *Agent) Activate() {
	a.IsActive = true
	a.UpdatedAt = time.Now()
}

// Deactivate deactivates the agent
func (a *Agent) Deactivate() {
	a.IsActive = false
	a.UpdatedAt = time.Now()
}
