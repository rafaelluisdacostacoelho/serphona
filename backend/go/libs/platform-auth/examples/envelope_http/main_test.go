//go:build examples
// +build examples

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReportsHandlerGET(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/reports", nil)
	rec := httptest.NewRecorder()

	reportsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReportsHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/reports", nil)
	rec := httptest.NewRecorder()

	reportsHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
