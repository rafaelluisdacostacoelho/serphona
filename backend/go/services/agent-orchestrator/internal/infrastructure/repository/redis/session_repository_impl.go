package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/entity"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/domain/repository"
)

// sessionRepositoryImpl implements repository.SessionRepository using Redis
type sessionRepositoryImpl struct {
	client *redis.Client
	ttl    time.Duration
}

// NewSessionRepository creates a new Redis session repository
func NewSessionRepository(client *redis.Client, ttl time.Duration) repository.SessionRepository {
	return &sessionRepositoryImpl{
		client: client,
		ttl:    ttl,
	}
}

// Redis key patterns
func sessionKey(id uuid.UUID) string {
	return fmt.Sprintf("session:%s", id.String())
}

func sessionMessagesKey(id uuid.UUID) string {
	return fmt.Sprintf("session:%s:messages", id.String())
}

func tenantSessionsKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("tenant:%s:sessions", tenantID.String())
}

// Create creates a new session
func (r *sessionRepositoryImpl) Create(ctx context.Context, session *entity.Session) error {
	// Serialize session to JSON
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store in Redis with TTL
	key := sessionKey(session.ID)
	if err := r.client.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store session in Redis: %w", err)
	}

	// Add to tenant sessions set
	tenantKey := tenantSessionsKey(session.TenantID)
	if err := r.client.SAdd(ctx, tenantKey, session.ID.String()).Err(); err != nil {
		return fmt.Errorf("failed to add session to tenant set: %w", err)
	}

	return nil
}

// FindByID finds a session by ID
func (r *sessionRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.Session, error) {
	key := sessionKey(id)

	data, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session from Redis: %w", err)
	}

	var session entity.Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// Update updates a session
func (r *sessionRepositoryImpl) Update(ctx context.Context, session *entity.Session) error {
	// Serialize session to JSON
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Update in Redis, maintaining existing TTL
	key := sessionKey(session.ID)
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to get TTL: %w", err)
	}

	// If key doesn't exist or has no expiry, use default TTL
	if ttl == -2 || ttl == -1 {
		ttl = r.ttl
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to update session in Redis: %w", err)
	}

	return nil
}

// Delete deletes a session
func (r *sessionRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	// Get session first to get tenant ID
	session, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete session key
	key := sessionKey(id)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session from Redis: %w", err)
	}

	// Delete messages
	messagesKey := sessionMessagesKey(id)
	if err := r.client.Del(ctx, messagesKey).Err(); err != nil {
		// Log error but don't fail
	}

	// Remove from tenant sessions set
	tenantKey := tenantSessionsKey(session.TenantID)
	if err := r.client.SRem(ctx, tenantKey, id.String()).Err(); err != nil {
		// Log error but don't fail
	}

	return nil
}

// FindByTenant finds all sessions for a tenant
func (r *sessionRepositoryImpl) FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*entity.Session, error) {
	tenantKey := tenantSessionsKey(tenantID)

	sessionIDs, err := r.client.SMembers(ctx, tenantKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant sessions: %w", err)
	}

	sessions := make([]*entity.Session, 0, len(sessionIDs))
	for _, idStr := range sessionIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}

		session, err := r.FindByID(ctx, id)
		if err != nil {
			continue // Session might have expired
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// FindByUser finds all sessions for a user
func (r *sessionRepositoryImpl) FindByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*entity.Session, error) {
	allSessions, err := r.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	userSessions := make([]*entity.Session, 0)
	for _, session := range allSessions {
		if session.UserID == userID {
			userSessions = append(userSessions, session)
		}
	}

	return userSessions, nil
}

// FindActive finds all active sessions
func (r *sessionRepositoryImpl) FindActive(ctx context.Context) ([]*entity.Session, error) {
	// Scan all session keys
	var cursor uint64
	var sessions []*entity.Session

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, "session:*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan sessions: %w", err)
		}

		for _, key := range keys {
			// Skip message keys
			if len(key) > 8 && key[len(key)-9:] == ":messages" {
				continue
			}

			data, err := r.client.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var session entity.Session
			if err := json.Unmarshal([]byte(data), &session); err != nil {
				continue
			}

			if session.IsActive() {
				sessions = append(sessions, &session)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return sessions, nil
}

// FindExpired finds all expired sessions
func (r *sessionRepositoryImpl) FindExpired(ctx context.Context) ([]*entity.Session, error) {
	// Scan all session keys
	var cursor uint64
	var sessions []*entity.Session

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, "session:*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan sessions: %w", err)
		}

		for _, key := range keys {
			// Skip message keys
			if len(key) > 8 && key[len(key)-9:] == ":messages" {
				continue
			}

			data, err := r.client.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var session entity.Session
			if err := json.Unmarshal([]byte(data), &session); err != nil {
				continue
			}

			if session.IsExpired() {
				sessions = append(sessions, &session)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return sessions, nil
}

// AddMessage adds a message to a session
func (r *sessionRepositoryImpl) AddMessage(ctx context.Context, sessionID uuid.UUID, message *entity.Message) error {
	// Serialize message
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Add to messages list
	key := sessionMessagesKey(sessionID)
	if err := r.client.RPush(ctx, key, data).Err(); err != nil {
		return fmt.Errorf("failed to add message to Redis: %w", err)
	}

	// Set TTL on messages list (same as session)
	if err := r.client.Expire(ctx, key, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL on messages: %w", err)
	}

	return nil
}

// GetMessages retrieves messages for a session
func (r *sessionRepositoryImpl) GetMessages(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*entity.Message, error) {
	key := sessionMessagesKey(sessionID)

	// Calculate range
	start := int64(offset)
	stop := int64(offset + limit - 1)

	// Get messages from list
	results, err := r.client.LRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get messages from Redis: %w", err)
	}

	messages := make([]*entity.Message, 0, len(results))
	for _, data := range results {
		var message entity.Message
		if err := json.Unmarshal([]byte(data), &message); err != nil {
			continue // Skip invalid messages
		}
		messages = append(messages, &message)
	}

	return messages, nil
}

// CountMessages counts messages in a session
func (r *sessionRepositoryImpl) CountMessages(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	key := sessionMessagesKey(sessionID)

	count, err := r.client.LLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}
