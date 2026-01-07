package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/service"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_TenantHeaderPropagation(t *testing.T) {
	var capturedTenant string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenant = r.Header.Get(middleware.TenantIDHeader)
		w.Header().Set(middleware.TenantIDHeader, capturedTenant)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	tool := &entity.Tool{
		ID:                uuid.New(),
		Name:              "http-test",
		DisplayName:       "http-test",
		Method:            entity.HTTPMethodGET,
		BaseURL:           server.URL,
		EndpointPath:      "/api/test",
		Headers:           json.RawMessage(`{}`),
		AuthType:          entity.AuthTypeNone.String(),
		InputSchema:       json.RawMessage(`{}`),
		OutputSchema:      json.RawMessage(`{}`),
		MaxRetries:        1,
		RetryDelaySeconds: 0,
	}

	client := service.NewHTTPClient(5*time.Second, "tools-gateway", "int", "")
	ctx := middleware.WithTenantID(context.Background(), "tenant-http-int")

	_, err := client.Execute(ctx, tool, map[string]interface{}{}, nil)
	require.NoError(t, err)
	require.Equal(t, "tenant-http-int", capturedTenant)
}

func TestGraphQLClient_TenantHeaderPropagation(t *testing.T) {
	var capturedTenant string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenant = r.Header.Get(middleware.TenantIDHeader)
		w.Header().Set(middleware.TenantIDHeader, capturedTenant)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	defer server.Close()

	integration := &entity.Integration{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "graphql-test",
		Type:     entity.IntegrationTypeGraphQL,
		BaseURL:  server.URL,
		GraphQLConfig: &entity.GraphQLConfig{
			Endpoint: "/graphql",
		},
	}

	client := service.NewGraphQLClient(5 * time.Second)
	ctx := middleware.WithTenantID(context.Background(), "tenant-graphql-int")

	_, err := client.ExecuteQuery(ctx, integration, "query { ok }", nil, "")
	require.NoError(t, err)
	require.Equal(t, "tenant-graphql-int", capturedTenant)
}

func TestSOAPClient_TenantHeaderPropagation(t *testing.T) {
	var capturedTenant string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenant = r.Header.Get(middleware.TenantIDHeader)
		w.Header().Set(middleware.TenantIDHeader, capturedTenant)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?>
<Envelope xmlns="http://schemas.xmlsoap.org/soap/envelope/">
  <Body>
    <Response>Success</Response>
  </Body>
</Envelope>`))
	}))
	defer server.Close()

	integration := &entity.Integration{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "soap-test",
		Type:     entity.IntegrationTypeSOAP,
		BaseURL:  server.URL,
		SOAPConfig: &entity.SOAPConfig{
			Namespace:   "http://schemas.xmlsoap.org/soap/envelope/",
			ServiceName: "svc",
			SOAPVersion: "1.1",
		},
	}

	client := service.NewSOAPClient(5 * time.Second)
	ctx := middleware.WithTenantID(context.Background(), "tenant-soap-int")

	_, err := client.Call(ctx, integration, "TestOperation", map[string]interface{}{}, "")
	require.NoError(t, err)
	require.Equal(t, "tenant-soap-int", capturedTenant)
}
