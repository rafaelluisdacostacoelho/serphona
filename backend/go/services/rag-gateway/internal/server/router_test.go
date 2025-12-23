package server

import (
	"bytes"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/server/handler"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/usecase/stub"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rag := handler.NewRAGHandler(stub.Ingest{}, stub.Query{}, stub.Namespace{})
	router := NewRouter(rag)

	req := httptest.NewRequest(nethttp.MethodGet, "/health", nil)
	rw := httptest.NewRecorder()

	router.ServeHTTP(rw, req)

	if rw.Code != nethttp.StatusOK {
		t.Fatalf("expected status 200, got %d", rw.Code)
	}
}

func TestAPIHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rag := handler.NewRAGHandler(stub.Ingest{}, stub.Query{}, stub.Namespace{})
	router := NewRouter(rag)

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/healthz", nil)
	rw := httptest.NewRecorder()

	router.ServeHTTP(rw, req)

	if rw.Code != nethttp.StatusOK {
		t.Fatalf("expected status 200, got %d", rw.Code)
	}
}

func TestIngestInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rag := handler.NewRAGHandler(stub.Ingest{}, stub.Query{}, stub.Namespace{})
	router := NewRouter(rag)

	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/ingest", bytes.NewBufferString("{invalid"))
	rw := httptest.NewRecorder()

	router.ServeHTTP(rw, req)

	if rw.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rw.Code)
	}
}

func TestIngestNotImplemented(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rag := handler.NewRAGHandler(stub.Ingest{}, stub.Query{}, stub.Namespace{})
	router := NewRouter(rag)

	payload := `{"tenant_id":"t1","namespace":"ns","content":"c"}`
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/ingest", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()

	router.ServeHTTP(rw, req)

	if rw.Code != nethttp.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", rw.Code)
	}
}

func TestIngestMissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rag := handler.NewRAGHandler(stub.Ingest{}, stub.Query{}, stub.Namespace{})
	router := NewRouter(rag)

	payload := `{"tenant_id":"t1"}`
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/ingest", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()

	router.ServeHTTP(rw, req)

	if rw.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rw.Code)
	}
}

func TestQueryNotImplemented(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rag := handler.NewRAGHandler(stub.Ingest{}, stub.Query{}, stub.Namespace{})
	router := NewRouter(rag)

	payload := `{"tenant_id":"t1","namespace":"ns","query":"q"}`
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/query", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()

	router.ServeHTTP(rw, req)

	if rw.Code != nethttp.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", rw.Code)
	}
}

func TestNamespacesNotImplemented(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rag := handler.NewRAGHandler(stub.Ingest{}, stub.Query{}, stub.Namespace{})
	router := NewRouter(rag)

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/namespaces", nil)
	rw := httptest.NewRecorder()

	router.ServeHTTP(rw, req)

	if rw.Code != nethttp.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", rw.Code)
	}
}
