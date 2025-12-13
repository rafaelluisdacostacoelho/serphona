// Package apikey contains application-level API key use cases.
package apikey

import (
	"context"
	"fmt"
	"time"

	"tenant-manager/internal/domain/apikey"
	"tenant-manager/internal/domain/events"

	"github.com/google/uuid"
)

// EventPublisher defines the interface for publishing events.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, data []byte) error
}

// Cache defines the interface for caching operations.
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// Service handles application-level API key operations.
type Service struct {
	domainService  *apikey.Service
	eventPublisher EventPublisher
	cache          Cache
}

// NewService creates a new application service.
func NewService(domainService *apikey.Service, eventPublisher EventPublisher, cache Cache) *Service {
	return &Service{
		domainService:  domainService,
		eventPublisher: eventPublisher,
		cache:          cache,
	}
}

// CreateAPIKey creates a new API key and publishes event.
func (s *Service) CreateAPIKey(ctx context.Context, tenantID uuid.UUID, name string, createdBy uuid.UUID, permissions []string, expiryDays int) (*apikey.APIKey, string, error) {
	// Create API key via domain service
	key, rawKey, err := s.domainService.Create(ctx, tenantID, name, createdBy, permissions, expiryDays)
	if err != nil {
		return nil, "", err
	}

	// Publish event
	if s.eventPublisher != nil {
		expiresAt := ""
		if key.ExpiresAt != nil {
			expiresAt = key.ExpiresAt.Format(time.RFC3339)
		}

		event := events.NewAPIKeyCreatedEvent(
			tenantID,
			key.ID,
			createdBy,
			key.Name,
			key.KeyPrefix,
			key.Permissions,
			expiresAt,
		)

		if err := s.publishEvent(ctx, event); err != nil {
			// Log error but don't fail the operation
			// In production, you might want to implement a retry mechanism
		}
	}

	// Cache the key (for faster authentication)
	if s.cache != nil {
		cacheKey := fmt.Sprintf("apikey:%s", key.KeyPrefix)
		if err := s.cache.Set(ctx, cacheKey, key.ID.String(), 24*time.Hour); err != nil {
			// Log error but don't fail
		}
	}

	return key, rawKey, nil
}

// RevokeAPIKey revokes an API key and publishes event.
func (s *Service) RevokeAPIKey(ctx context.Context, id uuid.UUID, revokedBy uuid.UUID) error {
	// Get key before revoking
	key, err := s.domainService.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Revoke via domain service
	if err := s.domainService.Revoke(ctx, id, revokedBy); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("apikey:%s", key.KeyPrefix)
		_ = s.cache.Delete(ctx, cacheKey)
	}

	// Publish event
	if s.eventPublisher != nil {
		event := events.NewAPIKeyRevokedEvent(
			key.TenantID,
			key.ID,
			revokedBy,
			key.Name,
			key.KeyPrefix,
			"",
		)

		if err := s.publishEvent(ctx, event); err != nil {
			// Log error but don't fail
		}
	}

	return nil
}

// RevokeAllForTenant revokes all API keys for a tenant.
func (s *Service) RevokeAllForTenant(ctx context.Context, tenantID uuid.UUID, revokedBy uuid.UUID) error {
	// Get all keys before revoking
	keys, err := s.domainService.ListForTenant(ctx, tenantID, false)
	if err != nil {
		return err
	}

	// Revoke all via domain service
	if err := s.domainService.RevokeAllForTenant(ctx, tenantID, revokedBy); err != nil {
		return err
	}

	// Invalidate cache for all keys
	if s.cache != nil {
		for _, key := range keys {
			cacheKey := fmt.Sprintf("apikey:%s", key.KeyPrefix)
			_ = s.cache.Delete(ctx, cacheKey)
		}
	}

	// Publish events for each key
	if s.eventPublisher != nil {
		for _, key := range keys {
			if key.Status == apikey.StatusActive {
				event := events.NewAPIKeyRevokedEvent(
					tenantID,
					key.ID,
					revokedBy,
					key.Name,
					key.KeyPrefix,
					"tenant revoked all keys",
				)
				_ = s.publishEvent(ctx, event)
			}
		}
	}

	return nil
}

// AuthenticateAPIKey authenticates an API key and records usage.
func (s *Service) AuthenticateAPIKey(ctx context.Context, rawKey string, ipAddress string) (*apikey.APIKey, error) {
	// Authenticate via domain service
	key, err := s.domainService.Authenticate(ctx, rawKey, ipAddress)
	if err != nil {
		// Record failed authentication
		if s.eventPublisher != nil {
			// Try to get key prefix for event
			prefix := apikey.ParseKeyPrefix(rawKey)
			event := events.NewAPIKeyUsedEvent(
				uuid.Nil, // Don't know tenant ID yet
				uuid.Nil, // Don't know key ID yet
				prefix,
				ipAddress,
				"",
				false,
				err.Error(),
			)
			_ = s.publishEvent(ctx, event)
		}
		return nil, err
	}

	// Record successful usage
	if err := s.domainService.RecordUsage(ctx, key.ID, ipAddress, "", true); err != nil {
		// Log error but don't fail authentication
	}

	// Publish usage event
	if s.eventPublisher != nil {
		event := events.NewAPIKeyUsedEvent(
			key.TenantID,
			key.ID,
			key.KeyPrefix,
			ipAddress,
			"",
			true,
			"",
		)
		_ = s.publishEvent(ctx, event)
	}

	return key, nil
}

// GetAPIKey gets an API key by ID with cache.
func (s *Service) GetAPIKey(ctx context.Context, id uuid.UUID) (*apikey.APIKey, error) {
	// Try cache first
	if s.cache != nil {
		cacheKey := fmt.Sprintf("apikey:id:%s", id.String())
		var cachedKey *apikey.APIKey
		if err := s.cache.Get(ctx, cacheKey, &cachedKey); err == nil && cachedKey != nil {
			return cachedKey, nil
		}
	}

	// Get from domain service
	key, err := s.domainService.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache for future requests
	if s.cache != nil {
		cacheKey := fmt.Sprintf("apikey:id:%s", id.String())
		_ = s.cache.Set(ctx, cacheKey, key, 1*time.Hour)
	}

	return key, nil
}

// ListAPIKeys lists API keys for a tenant.
func (s *Service) ListAPIKeys(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]*apikey.APIKey, error) {
	return s.domainService.ListForTenant(ctx, tenantID, activeOnly)
}

// DeleteAPIKey deletes an API key.
func (s *Service) DeleteAPIKey(ctx context.Context, id uuid.UUID) error {
	// Get key before deleting
	key, err := s.domainService.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete via domain service
	if err := s.domainService.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("apikey:%s", key.KeyPrefix)
		_ = s.cache.Delete(ctx, cacheKey)

		cacheKey = fmt.Sprintf("apikey:id:%s", id.String())
		_ = s.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// UpdateMetadata updates API key metadata.
func (s *Service) UpdateMetadata(ctx context.Context, id uuid.UUID, metadata apikey.Metadata) error {
	// Update via domain service
	if err := s.domainService.UpdateMetadata(ctx, id, metadata); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("apikey:id:%s", id.String())
		_ = s.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// AddIPToWhitelist adds an IP to the whitelist.
func (s *Service) AddIPToWhitelist(ctx context.Context, id uuid.UUID, ip string) error {
	if err := s.domainService.AddIPToWhitelist(ctx, id, ip); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("apikey:id:%s", id.String())
		_ = s.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// RemoveIPFromWhitelist removes an IP from the whitelist.
func (s *Service) RemoveIPFromWhitelist(ctx context.Context, id uuid.UUID, ip string) error {
	if err := s.domainService.RemoveIPFromWhitelist(ctx, id, ip); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("apikey:id:%s", id.String())
		_ = s.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// CheckPermission checks if an API key has a specific permission.
func (s *Service) CheckPermission(key *apikey.APIKey, permission string) error {
	return s.domainService.CheckPermission(key, permission)
}

// CountAPIKeys counts API keys for a tenant.
func (s *Service) CountAPIKeys(ctx context.Context, tenantID uuid.UUID, activeOnly bool) (int64, error) {
	return s.domainService.CountForTenant(ctx, tenantID, activeOnly)
}

// MarkExpiredKeys marks expired API keys and publishes events.
func (s *Service) MarkExpiredKeys(ctx context.Context, limit int) (int, error) {
	// Mark expired via domain service
	count, err := s.domainService.MarkExpired(ctx, limit)
	if err != nil {
		return 0, err
	}

	// Note: In a real implementation, you'd want to get the list of expired keys
	// and publish individual events for each one

	return count, nil
}

// publishEvent publishes an event to Kafka.
func (s *Service) publishEvent(ctx context.Context, event events.Event) error {
	if s.eventPublisher == nil {
		return nil
	}

	// Convert event to JSON
	data, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to Kafka
	topic := fmt.Sprintf("tenant-manager.%s", event.GetType())
	if err := s.eventPublisher.Publish(ctx, topic, data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// InvalidateCache invalidates all cached API keys for a tenant.
func (s *Service) InvalidateCache(ctx context.Context, tenantID uuid.UUID) error {
	if s.cache == nil {
		return nil
	}

	// Get all keys for tenant
	keys, err := s.domainService.ListForTenant(ctx, tenantID, false)
	if err != nil {
		return err
	}

	// Invalidate cache for each key
	for _, key := range keys {
		cacheKey := fmt.Sprintf("apikey:%s", key.KeyPrefix)
		_ = s.cache.Delete(ctx, cacheKey)

		cacheKey = fmt.Sprintf("apikey:id:%s", key.ID.String())
		_ = s.cache.Delete(ctx, cacheKey)
	}

	return nil
}
