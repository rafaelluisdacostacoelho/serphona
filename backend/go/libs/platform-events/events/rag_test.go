package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRAGIngestionRequestedEventJSON(t *testing.T) {
	evt := RAGIngestionRequestedEvent{
		TenantID:    "tenant-1",
		Namespace:   "support",
		Source:      "connector.s3",
		DocumentID:  "doc-1",
		Version:     "v2",
		ETag:        "etag123",
		URI:         "s3://bucket/key",
		Tags:        []string{"faq", "en"},
		ACL:         []string{"role:admin", "group:support"},
		TTLSeconds:  3600,
		Metadata:    map[string]string{"k": "v"},
		RequestedAt: time.Unix(1700000000, 0).UTC(),
	}

	b, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	required := []string{"tenant_id", "source", "document_id", "requested_at"}
	for _, k := range required {
		if _, ok := m[k]; !ok {
			t.Fatalf("expected key %s in json", k)
		}
	}
}
