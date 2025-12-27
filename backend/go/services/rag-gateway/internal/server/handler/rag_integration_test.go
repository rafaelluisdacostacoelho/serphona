package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/model"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/domain"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/server"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/server/handler"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/usecase/pgvector"
)

type memStore struct {
	chunks   []model.Chunk
	lastTopK int
}

func (m *memStore) UpsertChunks(_ context.Context, chunks []model.Chunk) error {
	for _, ch := range chunks {
		// simple upsert by chunk_id
		replaced := false
		for i, existing := range m.chunks {
			if existing.ChunkID == ch.ChunkID {
				m.chunks[i] = ch
				replaced = true
				break
			}
		}
		if !replaced {
			if ch.CreatedAt.IsZero() {
				ch.CreatedAt = time.Now().UTC()
			}
			m.chunks = append(m.chunks, ch)
		}
	}
	return nil
}

func (m *memStore) Query(_ context.Context, q model.Query) ([]model.Chunk, error) {
	m.lastTopK = q.TopK
	var filtered []model.Chunk
	for _, ch := range m.chunks {
		if ch.TenantID == q.TenantID && ch.Namespace == q.Namespace {
			filtered = append(filtered, ch)
		}
	}
	// limit to topK
	if q.TopK > 0 && len(filtered) > q.TopK {
		filtered = filtered[:q.TopK]
	}
	// set a deterministic score
	for i := range filtered {
		filtered[i].Score = 1 - float64(i)*0.01
	}
	return filtered, nil
}

func (m *memStore) Ping(_ context.Context) error { return nil }

type fakeEmbed struct{ vec []float32 }

func (f fakeEmbed) Embed(_ context.Context, inputs []string) ([][]float32, error) {
	out := make([][]float32, len(inputs))
	for i := range inputs {
		out[i] = f.vec
	}
	return out, nil
}

func TestIngestAndQueryEndToEndWithDefaults(t *testing.T) {
	store := &memStore{}
	embed := fakeEmbed{vec: []float32{1, 1, 1}}

	ingest := pgvector.NewIngestUsecase(store, embed, 3, nil)
	query := pgvector.NewQueryUsecase(store, embed, 5, 3)

	h := handler.NewRAGHandler(ingest, query, nil)
	router := server.NewRouter(h)

	// ingest
	ingestBody := map[string]any{
		"tenant_id": "t1",
		"namespace": "ns",
		"content":   "Reset your password in Settings > Security.",
		"metadata":  map[string]string{"document_id": "doc-42", "source": "faq"},
	}
	if err := doPost(router, "/api/v1/ingest", ingestBody, http.StatusAccepted); err != nil {
		t.Fatal(err)
	}

	if len(store.chunks) != 1 {
		t.Fatalf("expected 1 chunk stored, got %d", len(store.chunks))
	}

	// query without top_k to trigger handler default (5)
	queryBody := map[string]any{
		"tenant_id": "t1",
		"namespace": "ns",
		"query":     "how to reset password?",
		"filters":   map[string]string{"document_id": "doc-42"},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/query", bytes.NewReader(mustJSON(queryBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "t1")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp domain.QueryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response json: %v", err)
	}

	if got := len(resp.Results); got != 1 {
		t.Fatalf("expected 1 result, got %d", got)
	}
	r := resp.Results[0]
	if r.Content == "" || r.ChunkID == "" {
		t.Fatalf("expected content and chunk_id, got %+v", r)
	}
	if r.Metadata["document_id"] != "doc-42" {
		t.Fatalf("expected document_id doc-42, got %+v", r.Metadata)
	}
	if store.lastTopK != 5 {
		t.Fatalf("expected topK default 5 from handler, got %d", store.lastTopK)
	}
}

func doPost(router http.Handler, path string, body map[string]any, expected int) error {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(mustJSON(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "t1")
	router.ServeHTTP(w, req)
	if w.Code != expected {
		return fmt.Errorf("expected status %d, got %d", expected, w.Code)
	}
	return nil
}

func mustJSON(body map[string]any) []byte {
	b, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return b
}
