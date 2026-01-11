package handler

import (
	"encoding/json"
	"testing"
)

func TestValidateCreateRequestRejectsDeepSchema(t *testing.T) {
	req := baseCreateReq()
	req.InputSchema = json.RawMessage(`{"a":{"b":{"c":{"d":{"e":{"f":{"g":{"h":{"i":{}}}}}}}}}`)

	if err := validateCreateRequest(req); err == nil {
		t.Fatalf("expected validation error for deep input_schema")
	}
}

func TestValidateCreateRequestAcceptsValidPayload(t *testing.T) {
	req := baseCreateReq()
	if err := validateCreateRequest(req); err != nil {
		t.Fatalf("expected valid request, got: %v", err)
	}
}

func TestValidateCreateRequestRejectsInvalidAllowlistProtocol(t *testing.T) {
	req := baseCreateReq()
	req.Allowlist = json.RawMessage(`{"hosts":["api.example.com"],"protocols":["ftp"]}`)

	if err := validateCreateRequest(req); err == nil {
		t.Fatalf("expected validation error for invalid allowlist protocol")
	}
}

func TestValidateCreateRequestRejectsPayloadLimitTooHigh(t *testing.T) {
	req := baseCreateReq()
	req.PayloadLimit = maxPayloadBytesLimit + 1

	if err := validateCreateRequest(req); err == nil {
		t.Fatalf("expected validation error for payload_bytes_limit over cap")
	}
}

func TestValidateCreateRequestRejectsMaxRetriesTooHigh(t *testing.T) {
	req := baseCreateReq()
	req.MaxRetries = maxAllowedRetries + 1

	if err := validateCreateRequest(req); err == nil {
		t.Fatalf("expected validation error for max_retries over cap")
	}
}

func baseCreateReq() createToolRequest {
	return createToolRequest{
		Name:         "test",
		DisplayName:  "Test",
		Description:  "desc",
		Category:     nil,
		Tags:         []string{"a"},
		Version:      "1.0.0",
		Status:       "published",
		InputSchema:  json.RawMessage(`{"type":"object","properties":{"foo":{"type":"string"}}}`),
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"bar":{"type":"string"}}}`),
		Definition:   json.RawMessage(`{"kind":"http"}`),
		Allowlist:    json.RawMessage(`{"hosts":["api.example.com"],"protocols":["https"]}`),
		TimeoutSecs:  30,
		MaxRetries:   1,
		PayloadLimit: 1024,
		IsPublic:     true,
	}
}
