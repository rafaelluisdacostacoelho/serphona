package health

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerHealthy(t *testing.T) {
	handler := Handler(func() error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandlerLivenessFails(t *testing.T) {
	handler := Handler(func() error { return errors.New("boom") })

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestHandlerReadinessFails(t *testing.T) {
	handler := Handler(func() error { return nil }, func() error { return errors.New("deps down") })

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}
