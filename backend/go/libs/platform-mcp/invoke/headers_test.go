package invoke

import "testing"

func TestEnsureTenantHeaders(t *testing.T) {
	headers := EnsureTenantHeaders(nil, "t1", "svc")
	if headers["X-Tenant-ID"] != "t1" || headers["X-Service-ID"] != "svc" {
		t.Fatalf("headers not set: %+v", headers)
	}

	existing := map[string]string{"Authorization": "bearer"}
	mutated := EnsureTenantHeaders(existing, "t2", "")
	if mutated["X-Tenant-ID"] != "t2" || mutated["X-Service-ID"] != "" || mutated["Authorization"] != "bearer" {
		t.Fatalf("expected preserved and updated headers: %+v", mutated)
	}
}
