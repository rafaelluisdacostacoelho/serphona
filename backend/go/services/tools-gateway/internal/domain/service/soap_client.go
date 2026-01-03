// Package service contains domain services.
package service

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	authclient "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

// SOAPClient handles SOAP/WebService requests.
type SOAPClient interface {
	// Call executes a SOAP operation.
	Call(
		ctx context.Context,
		integration *entity.Integration,
		operation string,
		params map[string]interface{},
		token string,
	) (*SOAPResponse, error)

	// GetWSDL fetches the WSDL document.
	GetWSDL(
		ctx context.Context,
		integration *entity.Integration,
	) (string, error)
}

// SOAPResponse represents a SOAP response.
type SOAPResponse struct {
	Body       interface{}         `json:"body"`
	Headers    map[string][]string `json:"headers"`
	RawBody    []byte              `json:"-"`
	StatusCode int                 `json:"status_code"`
	LatencyMS  int64               `json:"latency_ms"`
}

// SOAPEnvelope represents a SOAP envelope structure.
type SOAPEnvelope struct {
	XMLName xml.Name    `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Header  *SOAPHeader `xml:"Header,omitempty"`
	Body    SOAPBody    `xml:"Body"`
}

// SOAPHeader represents SOAP headers.
type SOAPHeader struct {
	Items []interface{} `xml:",any"`
}

// SOAPBody represents the SOAP body.
type SOAPBody struct {
	Content interface{} `xml:",any"`
	Fault   *SOAPFault  `xml:"Fault,omitempty"`
}

// SOAPFault represents a SOAP fault.
type SOAPFault struct {
	XMLName xml.Name `xml:"Fault"`
	Code    string   `xml:"faultcode"`
	String  string   `xml:"faultstring"`
	Actor   string   `xml:"faultactor,omitempty"`
	Detail  string   `xml:"detail,omitempty"`
}

// soapClientImpl implements SOAPClient.
type soapClientImpl struct {
	httpClient *http.Client
}

// NewSOAPClient creates a new SOAPClient.
func NewSOAPClient(timeout time.Duration) SOAPClient {
	return &soapClientImpl{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: authclient.WithDefaultTransport(nil),
		},
	}
}

// Call executes a SOAP operation.
func (c *soapClientImpl) Call(
	ctx context.Context,
	integration *entity.Integration,
	operation string,
	params map[string]interface{},
	token string,
) (*SOAPResponse, error) {
	if integration.SOAPConfig == nil {
		return nil, fmt.Errorf("SOAP config is required")
	}

	startTime := time.Now()

	// Build SOAP envelope
	envelope, err := c.buildEnvelope(integration, operation, params)
	if err != nil {
		return nil, fmt.Errorf("failed to build envelope: %w", err)
	}

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SOAP envelope: %w", err)
	}

	// Add XML declaration
	xmlBody := []byte(xml.Header + string(xmlData))

	// Determine endpoint
	endpoint := integration.BaseURL
	if integration.SOAPConfig.ServiceName != "" {
		endpoint = fmt.Sprintf("%s/%s", endpoint, integration.SOAPConfig.ServiceName)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(xmlBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	c.addHeaders(req, integration, operation, token)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SOAP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse SOAP response
	response, err := c.parseResponse(body, resp.StatusCode)
	if err != nil {
		return nil, err
	}

	response.LatencyMS = time.Since(startTime).Milliseconds()
	response.Headers = resp.Header

	return response, nil
}

// GetWSDL fetches the WSDL document.
func (c *soapClientImpl) GetWSDL(
	ctx context.Context,
	integration *entity.Integration,
) (string, error) {
	if integration.SOAPConfig == nil {
		return "", fmt.Errorf("SOAP config is required")
	}

	wsdlURL := integration.SOAPConfig.WSDLURL
	if wsdlURL == "" {
		// Try appending ?wsdl to base URL
		wsdlURL = integration.BaseURL + "?wsdl"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", wsdlURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("WSDL request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read WSDL: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("WSDL request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

// buildEnvelope builds a SOAP envelope for the operation.
func (c *soapClientImpl) buildEnvelope(
	integration *entity.Integration,
	operation string,
	params map[string]interface{},
) (*SOAPEnvelope, error) {
	config := integration.SOAPConfig

	// Use custom envelope if provided
	if config.Envelope != "" {
		// Parse custom envelope template
		var envelope SOAPEnvelope
		if err := xml.Unmarshal([]byte(config.Envelope), &envelope); err != nil {
			return nil, fmt.Errorf("failed to parse custom envelope: %w", err)
		}
		return &envelope, nil
	}

	// Build standard envelope
	envelope := &SOAPEnvelope{
		Body: SOAPBody{
			Content: c.buildOperationBody(operation, params, config.Namespace),
		},
	}

	return envelope, nil
}

// buildOperationBody builds the operation body content.
func (c *soapClientImpl) buildOperationBody(
	operation string,
	params map[string]interface{},
	namespace string,
) map[string]interface{} {
	// Build operation element
	body := map[string]interface{}{
		"XMLName": xml.Name{
			Space: namespace,
			Local: operation,
		},
	}

	// Add parameters
	for key, value := range params {
		body[key] = value
	}

	return body
}

// parseResponse parses the SOAP response.
func (c *soapClientImpl) parseResponse(body []byte, statusCode int) (*SOAPResponse, error) {
	var envelope SOAPEnvelope
	if err := xml.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse SOAP response: %w", err)
	}

	// Check for SOAP fault
	if envelope.Body.Fault != nil {
		return &SOAPResponse{
			Body:       envelope.Body.Fault,
			RawBody:    body,
			StatusCode: statusCode,
		}, fmt.Errorf("SOAP fault: %s - %s", envelope.Body.Fault.Code, envelope.Body.Fault.String)
	}

	return &SOAPResponse{
		Body:       envelope.Body.Content,
		RawBody:    body,
		StatusCode: statusCode,
	}, nil
}

// addHeaders adds necessary headers to the request.
func (c *soapClientImpl) addHeaders(
	req *http.Request,
	integration *entity.Integration,
	operation string,
	token string,
) {
	config := integration.SOAPConfig

	// Set Content-Type based on SOAP version
	contentType := "text/xml; charset=utf-8"
	if config.SOAPVersion == "1.2" {
		contentType = "application/soap+xml; charset=utf-8"
	}
	req.Header.Set("Content-Type", contentType)

	// Add SOAPAction header (required for SOAP 1.1)
	if config.SOAPVersion != "1.2" {
		soapAction := fmt.Sprintf("%s/%s", config.Namespace, operation)
		req.Header.Set("SOAPAction", fmt.Sprintf(`"%s"`, soapAction))
	}

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	// Add tenant header from context (platform-auth)
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

// WSDLParser provides basic WSDL parsing functionality.
type WSDLParser struct{}

// ParseWSDL parses a WSDL document and extracts operations.
func (p *WSDLParser) ParseWSDL(wsdlContent string) ([]string, error) {
	// Basic WSDL parsing to extract operation names
	// This is a simplified version - a full implementation would use a proper WSDL parser

	operations := []string{}

	// Look for <operation> elements
	decoder := xml.NewDecoder(strings.NewReader(wsdlContent))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse WSDL: %w", err)
		}

		if se, ok := token.(xml.StartElement); ok {
			if se.Name.Local == "operation" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" {
						operations = append(operations, attr.Value)
					}
				}
			}
		}
	}

	return operations, nil
}
