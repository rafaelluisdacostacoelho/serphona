package model

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/metadata"
)

const (
	DefaultTopK     = 5
	DefaultMinScore = 0.0
)

// ChunkMetadata holds optional fields about a chunk/document.
type ChunkMetadata struct {
	Version    string            `json:"version,omitempty"`
	Source     string            `json:"source,omitempty"`
	URI        string            `json:"uri,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	ACL        []string          `json:"acl,omitempty"`
	TTLSeconds int               `json:"ttl_seconds,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Normalize dedupes/sanitizes fields and mirrors canonical values into Attributes.
func (m *ChunkMetadata) Normalize() {
	if m.Attributes == nil {
		m.Attributes = map[string]string{}
	}

	// Prefer explicit fields but allow Attributes as fallback inputs.
	if m.Version == "" {
		m.Version = m.Attributes["version"]
	}
	if m.Source == "" {
		m.Source = m.Attributes["source"]
	}
	if m.URI == "" {
		m.URI = m.Attributes["uri"]
	}

	m.Tags = dedupeStrings(m.Tags)
	m.ACL = dedupeStrings(m.ACL)

	if tagsAttr := m.Attributes["tags"]; tagsAttr != "" && len(m.Tags) == 0 {
		m.Tags = append(m.Tags, tagsAttr)
	}
	if aclAttr := m.Attributes["acl"]; aclAttr != "" && len(m.ACL) == 0 {
		m.ACL = append(m.ACL, aclAttr)
	}

	if m.Version != "" {
		m.Attributes["version"] = m.Version
	}
	if m.Source != "" {
		m.Attributes["source"] = m.Source
	}
	if m.URI != "" {
		m.Attributes["uri"] = m.URI
	}
	if len(m.Tags) > 0 {
		m.Attributes["tags"] = strings.Join(m.Tags, ",")
	}
	if len(m.ACL) > 0 {
		m.Attributes["acl"] = strings.Join(m.ACL, ",")
	}
}

// Validate checks basic constraints.
func (m ChunkMetadata) Validate() error {
	if m.TTLSeconds < 0 {
		return errors.New("ttl_seconds cannot be negative")
	}
	if m.TTLSeconds > metadata.MaxTTLSeconds {
		return fmt.Errorf("ttl_seconds exceeds max of %d", metadata.MaxTTLSeconds)
	}
	if m.URI != "" {
		if _, err := url.ParseRequestURI(m.URI); err != nil {
			return fmt.Errorf("uri is invalid: %w", err)
		}
	}
	return nil
}

// Chunk represents a stored fragment ready for retrieval.
type Chunk struct {
	TenantID   string        `json:"tenant_id"`
	Namespace  string        `json:"namespace"`
	DocumentID string        `json:"document_id"`
	ChunkID    string        `json:"chunk_id"`
	Content    string        `json:"content"`
	Metadata   ChunkMetadata `json:"metadata"`
	Embedding  []float32     `json:"embedding"`
	Score      float64       `json:"score"`
	ETag       string        `json:"etag"`
	CreatedAt  time.Time     `json:"created_at"`
}

// Filters constrain retrieval.
type Filters struct {
	Tags       []string          `json:"tags,omitempty"`
	ACL        []string          `json:"acl,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Source     string            `json:"source,omitempty"`
	Version    string            `json:"version,omitempty"`
	URI        string            `json:"uri,omitempty"`
	Language   string            `json:"language,omitempty"`
	Channel    string            `json:"channel,omitempty"`
}

// Normalize dedupes and trims strings, ensuring maps are non-nil.
func (f *Filters) Normalize() {
	if f.Attributes == nil {
		f.Attributes = map[string]string{}
	}
	f.Tags = dedupeStrings(f.Tags)
	f.ACL = dedupeStrings(f.ACL)
	f.Language = strings.TrimSpace(f.Language)
	f.Channel = strings.TrimSpace(f.Channel)
}

// Query represents a retrieval request against the vector store.
type Query struct {
	TenantID    string    `json:"tenant_id"`
	Namespace   string    `json:"namespace"`
	QueryVector []float32 `json:"query_vector"`
	TopK        int       `json:"top_k"`
	Filters     Filters   `json:"filters"`
	MinScore    float64   `json:"min_score"`
}

// Normalize applies sensible defaults.
func (q *Query) Normalize() {
	if q.TopK <= 0 {
		q.TopK = DefaultTopK
	}
	if q.MinScore < 0 {
		q.MinScore = DefaultMinScore
	}
	q.Filters.Normalize()
}

// Validate checks required fields and constraints.
func (q Query) Validate(expectedDim int) error {
	if strings.TrimSpace(q.TenantID) == "" {
		return errors.New("tenant_id is required")
	}
	if strings.TrimSpace(q.Namespace) == "" {
		return errors.New("namespace is required")
	}
	if len(q.QueryVector) == 0 {
		return errors.New("query_vector is required")
	}
	if expectedDim > 0 && len(q.QueryVector) != expectedDim {
		return fmt.Errorf("query_vector dimension mismatch: got %d, expected %d", len(q.QueryVector), expectedDim)
	}
	if q.MinScore < 0 {
		return errors.New("min_score cannot be negative")
	}
	return nil
}

// ValidateChunk enforces required multi-tenant fields and embedding dimension.
func ValidateChunk(ch Chunk, expectedDim int) error {
	if strings.TrimSpace(ch.TenantID) == "" || strings.TrimSpace(ch.Namespace) == "" || strings.TrimSpace(ch.ChunkID) == "" {
		return errors.New("tenant_id, namespace, and chunk_id are required")
	}
	if len(ch.Embedding) == 0 {
		return errors.New("embedding is required")
	}
	if expectedDim > 0 && len(ch.Embedding) != expectedDim {
		return fmt.Errorf("embedding dimension mismatch: got %d, expected %d", len(ch.Embedding), expectedDim)
	}
	if err := ch.Metadata.Validate(); err != nil {
		return err
	}
	return nil
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
