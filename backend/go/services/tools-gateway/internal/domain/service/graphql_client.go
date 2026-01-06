// Package service contains domain services.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	authclient "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

// GraphQLClient handles GraphQL API requests.
type GraphQLClient interface {
	// ExecuteQuery executes a GraphQL query.
	ExecuteQuery(
		ctx context.Context,
		integration *entity.Integration,
		query string,
		variables map[string]interface{},
		token string,
	) (*GraphQLResponse, error)

	// ExecuteMutation executes a GraphQL mutation.
	ExecuteMutation(
		ctx context.Context,
		integration *entity.Integration,
		mutation string,
		variables map[string]interface{},
		token string,
	) (*GraphQLResponse, error)

	// ExecuteBatch executes multiple GraphQL operations in a single request.
	ExecuteBatch(
		ctx context.Context,
		integration *entity.Integration,
		operations []GraphQLOperation,
		token string,
	) ([]GraphQLResponse, error)

	// IntrospectSchema fetches the GraphQL schema using introspection.
	IntrospectSchema(
		ctx context.Context,
		integration *entity.Integration,
		token string,
	) (map[string]interface{}, error)
}

// GraphQLOperation represents a single GraphQL operation.
type GraphQLOperation struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Operation string                 `json:"operationName,omitempty"`
}

// GraphQLResponse represents a GraphQL response.
type GraphQLResponse struct {
	Data       map[string]interface{} `json:"data,omitempty"`
	Errors     []GraphQLError         `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`

	// Metadata
	StatusCode int   `json:"-"`
	LatencyMS  int64 `json:"-"`
}

// GraphQLError represents a GraphQL error.
type GraphQLError struct {
	Message    string                 `json:"message"`
	Path       []interface{}          `json:"path,omitempty"`
	Locations  []GraphQLLocation      `json:"locations,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// GraphQLLocation represents an error location.
type GraphQLLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// graphQLClientImpl implements GraphQLClient.
type graphQLClientImpl struct {
	httpClient *http.Client
}

// NewGraphQLClient creates a new GraphQLClient.
func NewGraphQLClient(timeout time.Duration) GraphQLClient {
	return &graphQLClientImpl{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: authclient.WithDefaultTransport(nil),
		},
	}
}

// ExecuteQuery executes a GraphQL query.
func (c *graphQLClientImpl) ExecuteQuery(
	ctx context.Context,
	integration *entity.Integration,
	query string,
	variables map[string]interface{},
	token string,
) (*GraphQLResponse, error) {
	if integration.GraphQLConfig == nil {
		return nil, fmt.Errorf("GraphQL config is required")
	}

	operation := GraphQLOperation{
		Query:     query,
		Variables: variables,
	}

	return c.executeOperation(ctx, integration, operation, token)
}

// ExecuteMutation executes a GraphQL mutation.
func (c *graphQLClientImpl) ExecuteMutation(
	ctx context.Context,
	integration *entity.Integration,
	mutation string,
	variables map[string]interface{},
	token string,
) (*GraphQLResponse, error) {
	if integration.GraphQLConfig == nil {
		return nil, fmt.Errorf("GraphQL config is required")
	}

	operation := GraphQLOperation{
		Query:     mutation,
		Variables: variables,
	}

	return c.executeOperation(ctx, integration, operation, token)
}

// ExecuteBatch executes multiple GraphQL operations.
func (c *graphQLClientImpl) ExecuteBatch(
	ctx context.Context,
	integration *entity.Integration,
	operations []GraphQLOperation,
	token string,
) ([]GraphQLResponse, error) {
	if integration.GraphQLConfig == nil {
		return nil, fmt.Errorf("GraphQL config is required")
	}

	if !integration.GraphQLConfig.BatchingEnabled {
		return nil, fmt.Errorf("batching is not enabled for this integration")
	}

	startTime := time.Now()

	// Prepare batch request
	requestBody, err := json.Marshal(operations)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch request: %w", err)
	}

	endpoint := integration.BaseURL + integration.GraphQLConfig.Endpoint

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	c.addHeaders(req, integration, token)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("batch request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse batch response
	var responses []GraphQLResponse
	if err := json.Unmarshal(body, &responses); err != nil {
		return nil, fmt.Errorf("failed to parse batch response: %w", err)
	}

	latency := time.Since(startTime).Milliseconds()
	for i := range responses {
		responses[i].StatusCode = resp.StatusCode
		responses[i].LatencyMS = latency
	}

	return responses, nil
}

// IntrospectSchema fetches the GraphQL schema.
func (c *graphQLClientImpl) IntrospectSchema(
	ctx context.Context,
	integration *entity.Integration,
	token string,
) (map[string]interface{}, error) {
	if integration.GraphQLConfig == nil {
		return nil, fmt.Errorf("GraphQL config is required")
	}

	// Standard GraphQL introspection query
	introspectionQuery := `
		query IntrospectionQuery {
			__schema {
				queryType { name }
				mutationType { name }
				subscriptionType { name }
				types {
					...FullType
				}
				directives {
					name
					description
					locations
					args {
						...InputValue
					}
				}
			}
		}

		fragment FullType on __Type {
			kind
			name
			description
			fields(includeDeprecated: true) {
				name
				description
				args {
					...InputValue
				}
				type {
					...TypeRef
				}
				isDeprecated
				deprecationReason
			}
			inputFields {
				...InputValue
			}
			interfaces {
				...TypeRef
			}
			enumValues(includeDeprecated: true) {
				name
				description
				isDeprecated
				deprecationReason
			}
			possibleTypes {
				...TypeRef
			}
		}

		fragment InputValue on __InputValue {
			name
			description
			type { ...TypeRef }
			defaultValue
		}

		fragment TypeRef on __Type {
			kind
			name
			ofType {
				kind
				name
				ofType {
					kind
					name
					ofType {
						kind
						name
						ofType {
							kind
							name
						}
					}
				}
			}
		}
	`

	// Use introspection URL if available, otherwise use main endpoint
	// Note: endpoint selection is handled in executeOperation

	response, err := c.executeOperation(ctx, integration, GraphQLOperation{
		Query: introspectionQuery,
	}, token)

	if err != nil {
		return nil, err
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("introspection failed: %s", response.Errors[0].Message)
	}

	return response.Data, nil
}

// executeOperation executes a single GraphQL operation.
func (c *graphQLClientImpl) executeOperation(
	ctx context.Context,
	integration *entity.Integration,
	operation GraphQLOperation,
	token string,
) (*GraphQLResponse, error) {
	startTime := time.Now()

	// Merge with default variables
	if integration.GraphQLConfig.DefaultVariables != nil {
		if operation.Variables == nil {
			operation.Variables = make(map[string]interface{})
		}
		for key, value := range integration.GraphQLConfig.DefaultVariables {
			if _, exists := operation.Variables[key]; !exists {
				operation.Variables[key] = value
			}
		}
	}

	// Prepare request
	requestBody, err := json.Marshal(operation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := integration.BaseURL + integration.GraphQLConfig.Endpoint

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	c.addHeaders(req, integration, token)

	// Ensure tenant header
	tenantID, err := middleware.TenantIDFromContext(ctx)
	if err == nil && tenantID != "" {
		req.Header.Set(middleware.TenantIDHeader, tenantID)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GraphQL request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Debugging: Log response body
	fmt.Printf("GraphQL Response Body: %s\n", string(body))

	// Parse response
	var graphQLResp GraphQLResponse
	if err := json.Unmarshal(body, &graphQLResp); err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	graphQLResp.StatusCode = resp.StatusCode
	graphQLResp.LatencyMS = time.Since(startTime).Milliseconds()

	// Check for GraphQL errors
	if len(graphQLResp.Errors) > 0 {
		return &graphQLResp, fmt.Errorf("GraphQL errors: %s", graphQLResp.Errors[0].Message)
	}

	return &graphQLResp, nil
}

// addHeaders adds necessary headers to the request.
func (c *graphQLClientImpl) addHeaders(
	req *http.Request,
	integration *entity.Integration,
	token string,
) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	// Add tenant header using EnsureTenantHeaders
	if tenantID, err := middleware.TenantIDFromContext(req.Context()); err == nil && tenantID != "" {
		req.Header.Set(middleware.TenantIDHeader, tenantID)
	}

	// Add default headers
	if integration.DefaultHeaders != nil {
		for key, value := range integration.DefaultHeaders {
			req.Header.Set(key, value)
		}
	}
}
