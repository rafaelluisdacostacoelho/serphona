package customer

import (
	"testing"

	"github.com/google/uuid"
)

func TestCustomerTableName(t *testing.T) {
	if got := (Customer{}).TableName(); got != "customers" {
		t.Fatalf("unexpected table name %s", got)
	}
}

func TestNewCustomerAndMutators(t *testing.T) {
	tenantID := uuid.New()
	cust := NewCustomer(tenantID, "user@example.com", "Jane Doe")

	if cust.TenantID != tenantID || cust.Email != "user@example.com" || cust.Name != "Jane Doe" {
		t.Fatalf("unexpected customer fields %+v", cust)
	}
	if cust.Metadata == nil {
		t.Fatalf("expected metadata map to be initialized")
	}

	cust.SetStripeCustomerID("cus_123")
	if cust.StripeCustomerID != "cus_123" {
		t.Fatalf("expected stripe customer id to be set")
	}

	cust.Metadata = nil
	cust.UpdateMetadata("tier", "pro")
	if val, ok := cust.Metadata["tier"]; !ok || val != "pro" {
		t.Fatalf("expected metadata to be updated, got %+v", cust.Metadata)
	}
}
