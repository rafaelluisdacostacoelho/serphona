// Package redis provides Redis-based implementations.
package redis

import (
	"context"
	"fmt"
	"net/url"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NewClient creates a new Redis client from a URL.
func NewClient(ctx context.Context, redisURL, password string, db int, logger *zap.Logger) (*redis.Client, error) {
	// Parse Redis URL
	u, err := url.Parse(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	// Extract host and port
	addr := u.Host
	if addr == "" {
		addr = "localhost:6379"
	}

	// Override password if provided separately
	if password == "" && u.User != nil {
		password, _ = u.User.Password()
	}

	// Create client
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("redis client connected",
		zap.String("addr", addr),
		zap.Int("db", db),
	)

	return client, nil
}
