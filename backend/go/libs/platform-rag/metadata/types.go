package metadata

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const MaxTTLSeconds = 60 * 60 * 24 * 365

// DocumentMetadata centralizes document-level fields used across ingestion/indexing/retrieval.
type DocumentMetadata struct {
	TenantID   string            `json:"tenant_id"`
	Namespace  string            `json:"namespace"`
	DocumentID string            `json:"document_id"`
	Version    string            `json:"version,omitempty"`
	ETag       string            `json:"etag,omitempty"`
	Source     string            `json:"source,omitempty"`
	URI        string            `json:"uri,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	ACL        []string          `json:"acl,omitempty"`
	TTLSeconds int               `json:"ttl_seconds,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Normalize fills defaults from Attributes, deduplicates tags/acl, and mirrors canonical fields back into Attributes.
func (m *DocumentMetadata) Normalize() {
	if m.Attributes == nil {
		m.Attributes = map[string]string{}
	}

	// Prefer explicit fields but allow Attributes as fallback inputs.
	if m.DocumentID == "" {
		m.DocumentID = m.Attributes["document_id"]
	}
	if m.Version == "" {
		m.Version = m.Attributes["version"]
	}
	if m.ETag == "" {
		m.ETag = m.Attributes["etag"]
	}
	if m.Source == "" {
		m.Source = m.Attributes["source"]
	}
	if m.URI == "" {
		m.URI = m.Attributes["uri"]
	}

	m.Tags = dedupeStrings(m.Tags)
	m.ACL = dedupeStrings(m.ACL)

	// Mirror canonical values back to Attributes for downstream storage/query.
	if m.DocumentID != "" {
		m.Attributes["document_id"] = m.DocumentID
	}
	if m.Version != "" {
		m.Attributes["version"] = m.Version
	}
	if m.ETag != "" {
		m.Attributes["etag"] = m.ETag
	}
	if m.Source != "" {
		m.Attributes["source"] = m.Source
	}
	if m.URI != "" {
		m.Attributes["uri"] = m.URI
	}
}

// Validate enforces required multi-tenant fields and basic constraints.
func (m DocumentMetadata) Validate() error {
	if strings.TrimSpace(m.TenantID) == "" {
		return errors.New("tenant_id is required")
	}
	if strings.TrimSpace(m.Namespace) == "" {
		return errors.New("namespace is required")
	}
	if strings.TrimSpace(m.DocumentID) == "" {
		return errors.New("document_id is required")
	}
	if m.TTLSeconds < 0 {
		return errors.New("ttl_seconds cannot be negative")
	}
	if m.TTLSeconds > MaxTTLSeconds {
		return fmt.Errorf("ttl_seconds exceeds max of %d", MaxTTLSeconds)
	}
	if m.URI != "" {
		if _, err := url.ParseRequestURI(m.URI); err != nil {
			return fmt.Errorf("uri is invalid: %w", err)
		}
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
