package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/service"
)

// toolsClientImpl implements the ToolsClient interface using HTTP
type toolsClientImpl struct {
	baseURL    string
	httpClient *http.Client
}

// NewToolsClient creates a new Tools Gateway HTTP client
func NewToolsClient(baseURL string) service.ToolsClient {
	return &toolsClientImpl{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteTool executes a tool via HTTP request to Tools Gateway
func (c *toolsClientImpl) ExecuteTool(ctx context.Context, request *service.ToolExecutionRequest) (*service.ToolExecutionResponse, error) {
	// Prepare request body
	reqBody := map[string]interface{}{
		"tool_name":  request.ToolName,
		"parameters": request.Parameters,
		"tenant_id":  request.TenantID.String(),
		"user_id":    request.UserID,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/tools/execute", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("tool execution failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response service.ToolExecutionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAvailableTools retrieves the list of available tools for a tenant
func (c *toolsClientImpl) GetAvailableTools(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/tools?tenant_id=%s", c.baseURL, tenantID.String())
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get tools with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract tool names
	toolNames := make([]string, len(response.Tools))
	for i, tool := range response.Tools {
		toolNames[i] = tool.Name
	}

	return toolNames, nil
}

// ValidateTool checks if a tool is available and valid
func (c *toolsClientImpl) ValidateTool(ctx context.Context, tenantID uuid.UUID, toolName string) (bool, error) {
	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/tools/%s/validate?tenant_id=%s", c.baseURL, toolName, tenantID.String())
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	// Other error
	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("validation failed with status %d: %s", resp.StatusCode, string(body))
}
