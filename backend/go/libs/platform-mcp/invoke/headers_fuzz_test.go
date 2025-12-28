package invoke

import "testing"

func FuzzEnsureTenantHeaders(f *testing.F) {
	f.Add("tenant", "service")
	f.Add("", "")
	f.Fuzz(func(t *testing.T, tenant string, service string) {
		base := map[string]string{"foo": "bar"}
		out := EnsureTenantHeaders(base, tenant, service)
		if out == nil {
			t.Fatalf("nil headers returned")
		}
		if out["X-Tenant-ID"] != tenant {
			t.Fatalf("tenant header mismatch: %q vs %q", out["X-Tenant-ID"], tenant)
		}
		if service != "" && out["X-Service-ID"] != service {
			t.Fatalf("service header mismatch: %q vs %q", out["X-Service-ID"], service)
		}
		if base["foo"] != "bar" {
			t.Fatalf("existing header lost: %+v", base)
		}
	})
}
