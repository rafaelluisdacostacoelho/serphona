package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/middleware"
	"github.com/stretchr/testify/assert"
)

func TestTenantHeaderPropagation(t *testing.T) {
	assert := assert.New(t)

	// Mock server echoes tenant header back.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(middleware.TenantIDHeader, r.Header.Get(middleware.TenantIDHeader))
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, mockServer.URL, nil)
	assert.NoError(err)
	req.Header.Set(middleware.TenantIDHeader, "test-tenant")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(err)
	assert.Equal(http.StatusOK, resp.StatusCode)

	tenantHeader := resp.Header.Get(middleware.TenantIDHeader)
	assert.Equal("test-tenant", tenantHeader)
}
