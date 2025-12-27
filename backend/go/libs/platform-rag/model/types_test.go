package model

import (
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/metadata"
)

func TestQueryNormalizeAndValidate(t *testing.T) {
	q := Query{TenantID: "t1", Namespace: "ns", QueryVector: []float32{1, 2, 3}}
	q.Normalize()
	if q.TopK != DefaultTopK {
		t.Fatalf("expected default topk %d, got %d", DefaultTopK, q.TopK)
	}
	if q.MinScore != DefaultMinScore {
		t.Fatalf("expected default min score %f, got %f", DefaultMinScore, q.MinScore)
	}
	if err := q.Validate(3); err != nil {
		t.Fatalf("validate should pass: %v", err)
	}
	if err := q.Validate(4); err == nil {
		t.Fatalf("expected dimension mismatch error")
	}
}

func TestValidateChunk(t *testing.T) {
	ch := Chunk{
		TenantID:   "t1",
		Namespace:  "ns",
		DocumentID: "doc",
		ChunkID:    "c1",
		Content:    "content",
		Embedding:  []float32{1, 2, 3},
		Metadata:   ChunkMetadata{Source: "kb", Tags: []string{"faq", "faq"}},
	}
	ch.Metadata.Normalize()
	if err := ValidateChunk(ch, 3); err != nil {
		t.Fatalf("validate chunk failed: %v", err)
	}
	if len(ch.Metadata.Tags) != 1 {
		t.Fatalf("expected tags deduped, got %v", ch.Metadata.Tags)
	}

	ch.Embedding = nil
	if err := ValidateChunk(ch, 3); err == nil {
		t.Fatalf("expected embedding required error")
	}

	ch.Embedding = []float32{1, 2}
	if err := ValidateChunk(ch, 3); err == nil {
		t.Fatalf("expected embedding dimension error")
	}

	ch = Chunk{}
	if err := ValidateChunk(ch, 0); err == nil {
		t.Fatalf("expected required field error")
	}
}

func TestChunkMetadataNormalizeAndValidate(t *testing.T) {
	meta := ChunkMetadata{
		Attributes: map[string]string{
			"version": "v1",
			"source":  "kb",
			"uri":     "s3://bucket/key",
			"tags":    "faq",
			"acl":     "admin",
		},
		Tags: []string{"faq", ""},
		ACL:  []string{"admin", "admin"},
	}

	meta.Normalize()
	if err := meta.Validate(); err != nil {
		t.Fatalf("validate failed: %v", err)
	}

	if meta.Version != "v1" || meta.Source != "kb" || meta.URI != "s3://bucket/key" {
		t.Fatalf("canonical fields not populated from attributes: %+v", meta)
	}
	if len(meta.Tags) != 1 || meta.Tags[0] != "faq" {
		t.Fatalf("tags not deduped/filled: %+v", meta.Tags)
	}
	if len(meta.ACL) != 1 || meta.ACL[0] != "admin" {
		t.Fatalf("acl not deduped/filled: %+v", meta.ACL)
	}
	if meta.Attributes["tags"] == "" || meta.Attributes["acl"] == "" {
		t.Fatalf("tags/acl not mirrored into attributes: %+v", meta.Attributes)
	}

	meta.TTLSeconds = -1
	if err := meta.Validate(); err == nil {
		t.Fatalf("expected ttl negative error")
	}
}

func TestChunkMetadataValidateTTLUpperBoundAndURI(t *testing.T) {
	meta := ChunkMetadata{TTLSeconds: metadata.MaxTTLSeconds + 1}
	if err := meta.Validate(); err == nil {
		t.Fatalf("expected ttl upper bound validation error")
	}

	meta = ChunkMetadata{URI: "ht@tp//bad"}
	if err := meta.Validate(); err == nil {
		t.Fatalf("expected uri validation error")
	}
}

func TestChunkMetadataAttributeFallbacks(t *testing.T) {
	meta := ChunkMetadata{
		Attributes: map[string]string{
			"tags": "faq",
			"acl":  "admin",
		},
	}
	meta.Normalize()
	if len(meta.Tags) != 1 || meta.Tags[0] != "faq" {
		t.Fatalf("expected tags to be filled from attributes: %+v", meta.Tags)
	}
	if len(meta.ACL) != 1 || meta.ACL[0] != "admin" {
		t.Fatalf("expected acl to be filled from attributes: %+v", meta.ACL)
	}
	if meta.Attributes["tags"] == "" || meta.Attributes["acl"] == "" {
		t.Fatalf("expected attributes to be mirrored: %+v", meta.Attributes)
	}
}

func TestQueryValidateMinScore(t *testing.T) {
	q := Query{TenantID: "t", Namespace: "n", QueryVector: []float32{1}}
	q.MinScore = -0.1
	if err := q.Validate(1); err == nil {
		t.Fatalf("expected min_score validation error")
	}
}

func TestQueryNormalizeResetsMinScore(t *testing.T) {
	q := Query{TenantID: "t", Namespace: "n", QueryVector: []float32{1}, MinScore: -1}
	q.Normalize()
	if q.MinScore != DefaultMinScore {
		t.Fatalf("expected min score reset to default, got %f", q.MinScore)
	}
}

func TestQueryValidateTrimsWhitespace(t *testing.T) {
	q := Query{TenantID: " ", Namespace: "n", QueryVector: []float32{1}}
	if err := q.Validate(0); err == nil {
		t.Fatalf("expected tenant validation error with whitespace-only")
	}
}

func TestQueryValidateNamespaceWhitespace(t *testing.T) {
	q := Query{TenantID: "t", Namespace: " ", QueryVector: []float32{1}}
	if err := q.Validate(0); err == nil {
		t.Fatalf("expected namespace validation error with whitespace-only")
	}
}

func TestFiltersNormalizeTrimsLanguageChannel(t *testing.T) {
	f := Filters{Language: " en ", Channel: " email "}
	f.Normalize()
	if f.Language != "en" || f.Channel != "email" {
		t.Fatalf("expected trimmed language/channel, got %q/%q", f.Language, f.Channel)
	}
}

func TestFiltersNormalizeInitializesAttributesAndDedupe(t *testing.T) {
	f := Filters{Tags: []string{"a", "a"}, ACL: []string{"admin", "admin"}}
	f.Normalize()
	if f.Attributes == nil {
		t.Fatalf("attributes map should be initialized")
	}
	if len(f.Tags) != 1 || len(f.ACL) != 1 {
		t.Fatalf("expected deduped tags/acl, got %+v %+v", f.Tags, f.ACL)
	}
}

func TestValidateChunkFailsOnTTLAndETag(t *testing.T) {
	ch := Chunk{TenantID: "t", Namespace: "n", ChunkID: "c", Embedding: []float32{1}, Metadata: ChunkMetadata{TTLSeconds: -1}}
	if err := ValidateChunk(ch, 1); err == nil {
		t.Fatalf("expected ttl validation failure")
	}

	ch.Metadata = ChunkMetadata{}
	ch.Embedding = []float32{1, 2}
	if err := ValidateChunk(ch, 1); err == nil {
		t.Fatalf("expected dimension mismatch error")
	}
}

func TestQueryValidateRequiresVector(t *testing.T) {
	q := Query{TenantID: "t", Namespace: "n"}
	if err := q.Validate(0); err == nil {
		t.Fatalf("expected query_vector validation error")
	}
}
