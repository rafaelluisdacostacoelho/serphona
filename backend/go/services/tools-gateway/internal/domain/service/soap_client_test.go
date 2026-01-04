package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

func TestSOAPClientSetsTenantHeader(t *testing.T) {
	// Create a test server that captures the request
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(`<?xml version="1.0"?>
<Envelope xmlns="http://schemas.xmlsoap.org/soap/envelope/">
	<Body>
		<Response xmlns="http://schemas.xmlsoap.org/soap/envelope/">Success</Response>
	</Body>
</Envelope>`))
	}))
	defer server.Close()

	// Create context with tenant ID
	ctx := middleware.WithTenantID(context.Background(), "tenant-123")

	// Create mock integration
	integration := &entity.Integration{
		BaseURL: server.URL,
		SOAPConfig: &entity.SOAPConfig{
			WSDLURL: server.URL,
		},
	}

	// Create SOAP client
	soapClient := &soapClientImpl{
		httpClient: &http.Client{},
	}

	// Define params using a map
	params := map[string]interface{}{
		"key": "value",
		"nested": map[string]interface{}{
			"field": "nested-value",
		},
	}

	// Call the SOAP client
	_, err := soapClient.Call(ctx, integration, "test-operation", params, "test-token")
	if err != nil {
		t.Fatalf("SOAP client call failed: %v", err)
	}

	// Verify tenant header was set
	if capturedHeaders.Get(middleware.TenantIDHeader) != "tenant-123" {
		t.Errorf("expected X-Tenant-ID header 'tenant-123', got '%s'", capturedHeaders.Get(middleware.TenantIDHeader))
	}
}
