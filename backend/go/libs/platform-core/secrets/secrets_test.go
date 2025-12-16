package secrets

import "testing"

func TestGetSuccess(t *testing.T) {
	t.Setenv("MY_SECRET", "value")

	val, err := Get("MY_SECRET")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if val != "value" {
		t.Fatalf("unexpected value: %s", val)
	}
}

func TestGetMissing(t *testing.T) {
	t.Setenv("MY_SECRET", "")

	if _, err := Get("MY_SECRET"); err == nil {
		t.Fatalf("expected error for missing secret")
	}
}

type fakeProvider struct {
	value string
	err   error
}

func (f fakeProvider) Get(key string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.value, nil
}

func TestSetProvider(t *testing.T) {
	SetProvider(fakeProvider{value: "ok"})
	val, err := Get("ANY")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if val != "ok" {
		t.Fatalf("unexpected value: %s", val)
	}
}
