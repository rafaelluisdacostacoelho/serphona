// Package redis provides Redis client implementations.
package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"tenant-manager/internal/config"
)

// NewClient creates a new Redis client and wraps it in Cache.
func NewClient(ctx context.Context, cfg config.RedisConfig) (*Cache, error) {
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	if cfg.Password != "" {
		opt.Password = cfg.Password
	}
	opt.DB = cfg.DB

	client := redis.NewClient(opt)

	// Ping to verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return NewCache(client), nil
}
