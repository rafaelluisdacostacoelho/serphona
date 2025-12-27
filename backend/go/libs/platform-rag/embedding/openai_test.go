package embedding

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestOpenAIAdapterTokenGuardAndDimension(t *testing.T) {
	client := &fakeOpenAIClient{}
	tok := &fakeTokenizer{count: 5}
	cfg := OpenAIConfig{Model: "text-embedding-3-small", MaxTokens: 10, ExpectedDim: 3, Timeout: 2 * time.Second, RetryMax: 2}

	adapter, err := NewOpenAIAdapter(client, tok, cfg)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	embs, err := adapter.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(embs) != 1 || len(embs[0]) != 3 {
		t.Fatalf("unexpected embeddings: %+v", embs)
	}

	tok.count = 20
	if _, err := adapter.Embed(context.Background(), []string{"too many tokens"}); err == nil {
		t.Fatalf("expected token limit error")
	}

	client.dim = 2
	tok.count = 5
	if _, err := adapter.Embed(context.Background(), []string{"bad dim"}); err == nil {
		t.Fatalf("expected dimension error")
	}
}

func TestOpenAIAdapterRetries(t *testing.T) {
	client := &fakeOpenAIClient{failures: 1}
	tok := &fakeTokenizer{count: 1}
	cfg := OpenAIConfig{Model: "m", RetryMax: 3, RetryBackoffMin: time.Millisecond, RetryBackoffMax: 2 * time.Millisecond}

	adapter, err := NewOpenAIAdapter(client, tok, cfg)
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	if _, err := adapter.Embed(context.Background(), []string{"hi"}); err != nil {
		t.Fatalf("embed after retry: %v", err)
	}

	if client.calls != 2 {
		t.Fatalf("expected 2 calls (1 failure then success), got %d", client.calls)
	}
}

func TestOpenAIAdapterValidation(t *testing.T) {
	tok := &fakeTokenizer{count: 1}
	if _, err := NewOpenAIAdapter(nil, tok, OpenAIConfig{Model: "m"}); err == nil {
		t.Fatalf("expected client required error")
	}
	client := &fakeOpenAIClient{}
	if _, err := NewOpenAIAdapter(client, nil, OpenAIConfig{Model: "m"}); err == nil {
		t.Fatalf("expected tokenizer required error")
	}
	if _, err := NewOpenAIAdapter(client, tok, OpenAIConfig{}); err == nil {
		t.Fatalf("expected model required error")
	}
}

func TestOpenAIAdapterHandlesTokenizerErrorAndEmptyInputs(t *testing.T) {
	client := &fakeOpenAIClient{}
	tok := &fakeTokenizer{count: 1}
	adapter, err := NewOpenAIAdapter(client, tok, OpenAIConfig{Model: "m"})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	// empty inputs should short-circuit with nil result
	res, err := adapter.Embed(context.Background(), nil)
	if err != nil || res != nil {
		t.Fatalf("expected nil result for empty inputs")
	}

	// tokenizer error propagates
	terr := errors.New("tokenizer failure")
	badTok := &errTokenizer{err: terr}
	adapter.tokenizer = badTok
	if _, err := adapter.Embed(context.Background(), []string{"x"}); !errors.Is(err, terr) {
		t.Fatalf("expected tokenizer error, got %v", err)
	}
}

func TestOpenAIAdapterRetryExhaustion(t *testing.T) {
	client := &fakeOpenAIClient{failures: 3}
	tok := &fakeTokenizer{count: 1}
	adapter, err := NewOpenAIAdapter(client, tok, OpenAIConfig{Model: "m", RetryMax: 2, RetryBackoffMin: time.Millisecond, RetryBackoffMax: time.Millisecond})
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	if _, err := adapter.Embed(context.Background(), []string{"x"}); err == nil {
		t.Fatalf("expected final error after retries")
	}
}

type errTokenizer struct{ err error }

func (e *errTokenizer) CountTokens(string) (int, error) { return 0, e.err }

// fakes

type fakeTokenizer struct{ count int }

func (f *fakeTokenizer) CountTokens(string) (int, error) { return f.count, nil }

type fakeOpenAIClient struct {
	dim      int
	failures int
	calls    int
}

func (f *fakeOpenAIClient) CreateEmbeddings(ctx context.Context, req OpenAIEmbeddingRequest) (OpenAIEmbeddingResponse, error) {
	f.calls++
	if f.failures > 0 {
		f.failures--
		return OpenAIEmbeddingResponse{}, errors.New("temporary")
	}
	d := f.dim
	if d == 0 {
		d = 3
	}
	resp := OpenAIEmbeddingResponse{Data: make([]struct{ Embedding []float32 }, len(req.Inputs))}
	for i := range resp.Data {
		resp.Data[i].Embedding = make([]float32, d)
	}
	return resp, nil
}
