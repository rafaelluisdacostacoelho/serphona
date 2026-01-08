package usecase

import (
	"encoding/json"
	"testing"
)

func TestEnforceMethodPolicyBlocksNotAllowed(t *testing.T) {
	svc := &toolExecutorServiceImpl{policy: ExecutionPolicy{AllowedMethods: []string{"GET"}}}

	if err := svc.enforceMethodPolicy("POST", "sample", "tenant1"); err == nil {
		t.Fatalf("expected POST to be rejected when only GET is allowed")
	}
}

func TestEnforceHeaderPolicyBlocksDeniedHeader(t *testing.T) {
	headersJSON, _ := json.Marshal(map[string]string{"X-Evil": "1", "X-Good": "ok"})
	svc := &toolExecutorServiceImpl{policy: ExecutionPolicy{BlockedHeaders: []string{"x-evil"}}}

	if err := svc.enforceHeaderPolicy(headersJSON, "tool", "tenant"); err == nil {
		t.Fatalf("expected blocked header to be rejected")
	}
}

func TestEnforceQueryParamLimit(t *testing.T) {
	svc := &toolExecutorServiceImpl{policy: ExecutionPolicy{MaxQueryParams: 1}}

	if err := svc.enforceQueryParamLimit(map[string]interface{}{"a": 1, "b": 2}, "GET", "tool", "tenant"); err == nil {
		t.Fatalf("expected query param limit to trigger")
	}
}
