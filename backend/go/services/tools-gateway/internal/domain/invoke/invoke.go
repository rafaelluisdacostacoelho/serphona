package invoke

import "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/middleware"

// EnsureTenantHeaders ensures the tenant header is present in the map.
func EnsureTenantHeaders(headers map[string]string, tenantID, serviceName string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}
	if tenantID != "" {
		headers[middleware.TenantIDHeader] = tenantID
	}
	return headers
}
