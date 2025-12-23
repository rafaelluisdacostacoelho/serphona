package stub

import (
	"context"
	"errors"
)

// Client is a stub embedding client that always errors.
type Client struct{}

func (Client) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, errors.New("embedding provider not configured")
}
