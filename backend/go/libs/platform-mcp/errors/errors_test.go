package errors

import "testing"

func TestNewErrorAndString(t *testing.T) {
	err := New(ErrInvalidRequest, "bad input")
	if err.Code != ErrInvalidRequest {
		t.Fatalf("unexpected code: %s", err.Code)
	}
	if got := err.Error(); got != "bad input" {
		t.Fatalf("unexpected message: %s", got)
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrUnsupportedVersionSentinel.Error() != string(ErrUnsupportedVersion) {
		t.Fatalf("unexpected unsupported version sentinel")
	}
	if ErrMissingTenantSentinel.Error() != string(ErrMissingTenant) {
		t.Fatalf("unexpected missing tenant sentinel")
	}
	if ErrPolicyDeniedSentinel.Error() != string(ErrPolicyDenied) {
		t.Fatalf("unexpected policy denied sentinel")
	}
}
