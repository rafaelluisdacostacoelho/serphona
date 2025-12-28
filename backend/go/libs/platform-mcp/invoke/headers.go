package invoke

// EnsureTenantHeaders sets required tenant and service identity headers for outbound calls.
// It mutates or creates the map and returns it for chaining.
func EnsureTenantHeaders(headers map[string]string, tenantID, serviceID string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["X-Tenant-ID"] = tenantID
	if serviceID != "" {
		headers["X-Service-ID"] = serviceID
	}
	return headers
}
