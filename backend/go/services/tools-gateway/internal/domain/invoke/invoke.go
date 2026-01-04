package invoke

// EnsureTenantHeaders ensures the X-Tenant-ID header is set in the headers map
func EnsureTenantHeaders(headers map[string]string, tenantID, serviceName string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}
	if tenantID != "" {
		headers["X-Tenant-ID"] = tenantID
	}
	return headers
}
