package metadata

import "testing"

func TestNormalizeAndValidate(t *testing.T) {
	meta := DocumentMetadata{
		TenantID:  "t1",
		Namespace: "ns",
		Attributes: map[string]string{
			"document_id": "doc-1",
			"version":     "v1",
			"etag":        "e1",
			"source":      "web",
			"uri":         "s3://bucket/key",
		},
		Tags: []string{"faq", "faq", ""},
		ACL:  []string{"admin", "admin"},
	}

	meta.Normalize()
	if err := meta.Validate(); err != nil {
		t.Fatalf("validate failed: %v", err)
	}

	if meta.DocumentID != "doc-1" || meta.Version != "v1" || meta.ETag != "e1" || meta.Source != "web" || meta.URI != "s3://bucket/key" {
		t.Fatalf("fallback fields not populated: %+v", meta)
	}

	if len(meta.Tags) != 1 || meta.Tags[0] != "faq" {
		t.Fatalf("tags not deduped: %+v", meta.Tags)
	}
	if len(meta.ACL) != 1 || meta.ACL[0] != "admin" {
		t.Fatalf("acl not deduped: %+v", meta.ACL)
	}

	if meta.Attributes["document_id"] != "doc-1" || meta.Attributes["version"] != "v1" || meta.Attributes["etag"] != "e1" || meta.Attributes["source"] != "web" || meta.Attributes["uri"] != "s3://bucket/key" {
		t.Fatalf("attributes not mirrored: %+v", meta.Attributes)
	}
}

func TestValidateFailsOnMissingFields(t *testing.T) {
	meta := DocumentMetadata{}
	meta.Normalize()
	if err := meta.Validate(); err == nil {
		t.Fatalf("expected validation error for missing tenant/namespace/document_id")
	}

	meta = DocumentMetadata{TenantID: "t1", Namespace: "ns", DocumentID: "doc", TTLSeconds: -1}
	meta.Normalize()
	if err := meta.Validate(); err == nil {
		t.Fatalf("expected validation error for ttl_seconds")
	}
}

func TestValidateFailsOnTTLUpperBoundAndURI(t *testing.T) {
	meta := DocumentMetadata{TenantID: "t1", Namespace: "ns", DocumentID: "doc", TTLSeconds: MaxTTLSeconds + 1}
	meta.Normalize()
	if err := meta.Validate(); err == nil {
		t.Fatalf("expected validation error for ttl_seconds upper bound")
	}

	meta = DocumentMetadata{TenantID: "t1", Namespace: "ns", DocumentID: "doc", URI: ":://bad uri"}
	meta.Normalize()
	if err := meta.Validate(); err == nil {
		t.Fatalf("expected validation error for bad uri")
	}
}

func TestDocumentMetadataValidateSuccess(t *testing.T) {
	meta := DocumentMetadata{
		TenantID:   "t1",
		Namespace:  "ns",
		DocumentID: "doc",
		TTLSeconds: 60,
		URI:        "https://example.com",
		Tags:       []string{"a"},
	}
	meta.Normalize()
	if err := meta.Validate(); err != nil {
		t.Fatalf("expected valid metadata, got %v", err)
	}
}

func TestValidateMissingNamespaceAndDocument(t *testing.T) {
	meta := DocumentMetadata{TenantID: "t1", Namespace: "   ", DocumentID: "doc"}
	meta.Normalize()
	if err := meta.Validate(); err == nil {
		t.Fatalf("expected namespace validation error")
	}

	meta = DocumentMetadata{TenantID: "t1", Namespace: "ns", DocumentID: "   "}
	meta.Normalize()
	if err := meta.Validate(); err == nil {
		t.Fatalf("expected document_id validation error")
	}
}
