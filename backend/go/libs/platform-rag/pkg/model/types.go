package model

import "time"

// Chunk represents a stored fragment ready for retrieval.
type Chunk struct {
	TenantID   string            `json:"tenant_id"`
	Namespace  string            `json:"namespace"`
	DocumentID string            `json:"document_id"`
	ChunkID    string            `json:"chunk_id"`
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata"`
	Embedding  []float32         `json:"embedding"`
	Score      float64           `json:"score"`
	ETag       string            `json:"etag"`
	CreatedAt  time.Time         `json:"created_at"`
}

// Query represents a retrieval request against the vector store.
type Query struct {
	TenantID    string            `json:"tenant_id"`
	Namespace   string            `json:"namespace"`
	QueryVector []float32         `json:"query_vector"`
	TopK        int               `json:"top_k"`
	Filters     map[string]string `json:"filters"`
	MinScore    float64           `json:"min_score"`
}
