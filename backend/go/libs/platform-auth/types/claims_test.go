package types

import "testing"

func TestClaimsValidUser(t *testing.T) {
	c := Claims{
		TenantID: "11111111-1111-1111-1111-111111111111",
		UserID:   "22222222-2222-2222-2222-222222222222",
		Role:     "admin",
	}

	if err := c.Valid(); err != nil {
		t.Fatalf("expected valid claims, got %v", err)
	}
}

func TestClaimsValidService(t *testing.T) {
	c := Claims{
		TenantID: "platform",
		Service:  "tools-gateway",
	}

	if err := c.Valid(); err != nil {
		t.Fatalf("expected valid service claims, got %v", err)
	}
}

func TestClaimsInvalidCases(t *testing.T) {
	tests := []Claims{
		{UserID: "22222222-2222-2222-2222-222222222222", Role: "user"},                                       // missing tenant
		{TenantID: "not-a-uuid", UserID: "22222222-2222-2222-2222-222222222222", Role: "user"},               // invalid tenant
		{TenantID: "platform", UserID: "22222222-2222-2222-2222-222222222222", Service: "svc", Role: "user"}, // user + service
		{TenantID: "platform"}, // neither user nor service
		{TenantID: "platform", UserID: "22222222-2222-2222-2222-222222222222", Role: "viewer"}, // invalid role
		{TenantID: "platform", UserID: "not-a-uuid", Role: "user"},                             // bad user uuid
	}

	for i, c := range tests {
		if err := c.Valid(); err == nil {
			t.Fatalf("case %d expected invalid claims", i)
		}
	}
}

func TestClaimHelpers(t *testing.T) {
	c := Claims{
		Role:   "superadmin",
		Scopes: []string{"read", "write"},
	}

	if !c.HasRole("superadmin") {
		t.Fatalf("expected HasRole true")
	}
	if !c.IsAdmin() {
		t.Fatalf("expected IsAdmin true")
	}
	if !c.IsSuperAdmin() {
		t.Fatalf("expected IsSuperAdmin true")
	}
	if !c.HasScope("read") {
		t.Fatalf("expected HasScope true")
	}
	if !c.HasAnyScope("write", "other") {
		t.Fatalf("expected HasAnyScope true")
	}
	if c.HasAnyScope("none", "other") {
		t.Fatalf("expected HasAnyScope false")
	}
	if !c.HasAllScopes("read", "write") {
		t.Fatalf("expected HasAllScopes true")
	}
	if c.HasAllScopes("read", "missing") {
		t.Fatalf("expected HasAllScopes false")
	}

	user := Claims{Role: "user"}
	if user.IsAdmin() || user.IsSuperAdmin() {
		t.Fatalf("expected non-admin flags false")
	}
}
