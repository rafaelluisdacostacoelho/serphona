// Package redis provides Redis-based implementations.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"voice-gateway/internal/domain/call"
)

// CallStateRepository implements call state persistence using Redis.
type CallStateRepository struct {
	client *redis.Client
	ttl    time.Duration
}

// NewCallStateRepository creates a new Redis-based call state repository.
func NewCallStateRepository(client *redis.Client, ttl time.Duration) *CallStateRepository {
	return &CallStateRepository{
		client: client,
		ttl:    ttl,
	}
}

// Save stores a call state in Redis.
func (r *CallStateRepository) Save(ctx context.Context, c *call.Call) error {
	key := fmt.Sprintf("call:%s", c.ID)

	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal call: %w", err)
	}

	if err := r.client.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save call state: %w", err)
	}

	// Also index by channel ID for quick lookup
	channelKey := fmt.Sprintf("call:channel:%s", c.ChannelID)
	if err := r.client.Set(ctx, channelKey, c.ID.String(), r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save channel index: %w", err)
	}

	// Index by tenant for listing
	tenantKey := fmt.Sprintf("calls:tenant:%s", c.TenantID)
	if err := r.client.SAdd(ctx, tenantKey, c.ID.String()).Err(); err != nil {
		return fmt.Errorf("failed to add to tenant index: %w", err)
	}
	r.client.Expire(ctx, tenantKey, r.ttl)

	return nil
}

// Get retrieves a call by ID.
func (r *CallStateRepository) Get(ctx context.Context, callID uuid.UUID) (*call.Call, error) {
	key := fmt.Sprintf("call:%s", callID)

	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("call not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get call: %w", err)
	}

	var c call.Call
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal call: %w", err)
	}

	return &c, nil
}

// GetByChannelID retrieves a call by Asterisk channel ID.
func (r *CallStateRepository) GetByChannelID(ctx context.Context, channelID string) (*call.Call, error) {
	channelKey := fmt.Sprintf("call:channel:%s", channelID)

	callIDStr, err := r.client.Get(ctx, channelKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("call not found for channel")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get channel index: %w", err)
	}

	callID, err := uuid.Parse(callIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid call ID in index: %w", err)
	}

	return r.Get(ctx, callID)
}

// ListByTenant lists all active calls for a tenant.
func (r *CallStateRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*call.Call, error) {
	tenantKey := fmt.Sprintf("calls:tenant:%s", tenantID)

	callIDs, err := r.client.SMembers(ctx, tenantKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant calls: %w", err)
	}

	calls := make([]*call.Call, 0, len(callIDs))
	for _, idStr := range callIDs {
		callID, err := uuid.Parse(idStr)
		if err != nil {
			continue // Skip invalid IDs
		}

		c, err := r.Get(ctx, callID)
		if err != nil {
			continue // Skip calls that no longer exist
		}

		calls = append(calls, c)
	}

	return calls, nil
}

// Delete removes a call from Redis.
func (r *CallStateRepository) Delete(ctx context.Context, callID uuid.UUID) error {
	// Get call first to clean up indexes
	c, err := r.Get(ctx, callID)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("call:%s", callID)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete call: %w", err)
	}

	// Clean up channel index
	if c.ChannelID != "" {
		channelKey := fmt.Sprintf("call:channel:%s", c.ChannelID)
		r.client.Del(ctx, channelKey)
	}

	// Remove from tenant index
	tenantKey := fmt.Sprintf("calls:tenant:%s", c.TenantID)
	r.client.SRem(ctx, tenantKey, callID.String())

	return nil
}

// UpdateState updates only the state of a call (optimized).
func (r *CallStateRepository) UpdateState(ctx context.Context, callID uuid.UUID, state call.State) error {
	// For simplicity, we'll fetch, update, and save
	// In production, consider using Redis Hash for partial updates
	c, err := r.Get(ctx, callID)
	if err != nil {
		return err
	}

	c.State = state
	return r.Save(ctx, c)
}

// CountActive returns the number of active calls.
func (r *CallStateRepository) CountActive(ctx context.Context) (int64, error) {
	pattern := "call:*"
	var cursor uint64
	count := int64(0)

	for {
		var keys []string
		var err error
		keys, cursor, err = r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return 0, fmt.Errorf("failed to scan keys: %w", err)
		}

		for _, key := range keys {
			// Only count actual call keys, not indexes
			if len(key) > 36 && key[:5] == "call:" && key[5] != 'c' && key[5] != 't' {
				count++
			}
		}

		if cursor == 0 {
			break
		}
	}

	return count, nil
}
