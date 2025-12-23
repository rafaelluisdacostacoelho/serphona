package events

import "time"

// RAGIngestionRequestedEvent signals that a document should be ingested into the RAG pipeline.
type RAGIngestionRequestedEvent struct {
	TenantID    string            `json:"tenant_id"`
	Namespace   string            `json:"namespace,omitempty"`
	Source      string            `json:"source"`
	DocumentID  string            `json:"document_id"`
	Version     string            `json:"version,omitempty"`
	ETag        string            `json:"etag,omitempty"`
	URI         string            `json:"uri,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	ACL         []string          `json:"acl,omitempty"`
	TTLSeconds  int               `json:"ttl_seconds,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	RequestedAt time.Time         `json:"requested_at"`
}
