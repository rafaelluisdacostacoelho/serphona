package dto

// CreateRAGIngestionRequest is the payload for triggering ingestion via REST/GraphQL.
type CreateRAGIngestionRequest struct {
	Namespace  string            `json:"namespace" binding:"required"`
	DocumentID string            `json:"document_id" binding:"required"`
	URI        string            `json:"uri" binding:"required"`
	Source     string            `json:"source"`
	Version    string            `json:"version"`
	ETag       string            `json:"etag"`
	Tags       []string          `json:"tags"`
	ACL        []string          `json:"acl"`
	TTLSeconds *int              `json:"ttl_seconds"`
	Metadata   map[string]string `json:"metadata"`
}
