package domain

import "time"

// Namespace identifies a logical scope for RAG data per tenant.
type Namespace struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// NamespaceCreate captures creation payload.
type NamespaceCreate struct {
	Name string `json:"name" binding:"required"`
}

// IngestRequest carries a document or chunk to be indexed.
type IngestRequest struct {
	TenantID   string            `json:"tenant_id"`
	Namespace  string            `json:"namespace" binding:"required"`
	Content    string            `json:"content" binding:"required"`
	DocumentID string            `json:"document_id,omitempty"`
	Version    string            `json:"version,omitempty"`
	ETag       string            `json:"etag,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	ACL        []string          `json:"acl,omitempty"`
	TTLSeconds int               `json:"ttl_seconds,omitempty"`
	Metadata   map[string]string `json:"metadata"`
}

// QueryRequest carries a query for retrieval.
type QueryRequest struct {
	TenantID  string            `json:"tenant_id"`
	Namespace string            `json:"namespace" binding:"required"`
	Query     string            `json:"query" binding:"required"`
	TopK      int               `json:"top_k"`
	Filters   map[string]string `json:"filters"`
}

// QueryResult represents a retrieved item.
type QueryResult struct {
	DocumentID string            `json:"document_id,omitempty"`
	ChunkID    string            `json:"chunk_id"`
	Content    string            `json:"content"`
	Score      float64           `json:"score"`
	Metadata   map[string]string `json:"metadata"`
	ETag       string            `json:"etag,omitempty"`
}

// QueryResponse aggregates results.
type QueryResponse struct {
	Results []QueryResult `json:"results"`
}
