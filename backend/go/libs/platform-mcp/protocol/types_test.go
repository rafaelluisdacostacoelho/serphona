package protocol

import (
	"encoding/json"
	"testing"
)

func TestValidateTool(t *testing.T) {
	tool := Tool{
		Name:         "echo",
		Version:      "1.0.0",
		TenantID:     "tenant-1",
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`),
	}
	if err := ValidateTool(tool); err != nil {
		t.Fatalf("expected tool to be valid, got %v", err)
	}

	tool.TenantID = ""
	if err := ValidateTool(tool); err == nil {
		t.Fatalf("expected missing tenant error")
	}
}

func TestValidateInvocation(t *testing.T) {
	req := InvocationRequest{
		Version:  CurrentVersion,
		TenantID: "tenant-1",
		Tool:     ToolRef{Name: "echo"},
		Input:    json.RawMessage(`{"msg":"hi"}`),
	}
	if err := ValidateInvocation(req); err != nil {
		t.Fatalf("expected invocation to be valid, got %v", err)
	}

	req.Version = "v0"
	if err := ValidateInvocation(req); err == nil {
		t.Fatalf("expected unsupported version error")
	}
}
