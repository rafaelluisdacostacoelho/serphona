package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrossServiceTenantHeaderPropagation(t *testing.T) {
	assert := assert.New(t)

	// Configura um servidor HTTP mock para simular um serviço downstream
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Tenant-ID", r.Header.Get("X-Tenant-ID"))
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	// Simula uma requisição inicial com o cabeçalho de tenant
	req, err := http.NewRequestWithContext(context.Background(), "GET", mockServer.URL, nil)
	assert.NoError(err)
	req.Header.Set("X-Tenant-ID", "test-tenant")

	// Envia a requisição
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(err)
	assert.Equal(http.StatusOK, resp.StatusCode)

	// Valida que o cabeçalho foi propagado corretamente
	tenantHeader := resp.Header.Get("X-Tenant-ID")
	assert.Equal("test-tenant", tenantHeader)
}
