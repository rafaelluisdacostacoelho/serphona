package embedding

import "context"

// Client computes embeddings for given inputs.
type Client interface {
	Embed(ctx context.Context, inputs []string) ([][]float32, error)
}
