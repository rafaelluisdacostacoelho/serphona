package apikey

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"tenant-manager/internal/domain/apikey"
	"tenant-manager/internal/domain/tenant"

	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	authtypes "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

func TestRevokeAPIKeyInvalidatesCache(t *testing.T) {
	tenantID := uuid.New()
	createdBy := uuid.New()
	repo := newMemoryAPIKeyRepo()

	key, _, err := apikey.NewAPIKey(tenantID, "primary", createdBy, []string{apikey.PermissionAll})
	if err != nil {
		t.Fatalf("failed to seed key: %v", err)
	}

	if err := repo.Save(context.Background(), key); err != nil {
		t.Fatalf("failed to save key: %v", err)
	}

	svc := NewService(apikey.NewService(repo), nil, nil, &spyCache{})
	cache := svc.cache.(*spyCache)

	ctx := withTenantClaims(context.Background(), tenantID)
	revokedBy := uuid.New()

	if err := svc.RevokeAPIKey(ctx, key.ID, revokedBy); err != nil {
		t.Fatalf("RevokeAPIKey returned error: %v", err)
	}

	expected := []string{
		fmt.Sprintf("apikey:%s", key.KeyPrefix),
		fmt.Sprintf("apikey:id:%s", key.ID.String()),
	}

	assertDeletedKeys(t, cache.deleted, expected)
}

func TestRevokeAllForTenantInvalidatesCache(t *testing.T) {
	tenantID := uuid.New()
	createdBy := uuid.New()
	repo := newMemoryAPIKeyRepo()

	keyOne, _, _ := apikey.NewAPIKey(tenantID, "one", createdBy, []string{apikey.PermissionAll})
	keyTwo, _, _ := apikey.NewAPIKey(tenantID, "two", createdBy, []string{apikey.PermissionAll})
	_ = repo.Save(context.Background(), keyOne)
	_ = repo.Save(context.Background(), keyTwo)

	cache := &spyCache{}
	svc := NewService(apikey.NewService(repo), nil, nil, cache)

	ctx := withTenantClaims(context.Background(), tenantID)
	revokedBy := uuid.New()

	if err := svc.RevokeAllForTenant(ctx, tenantID, revokedBy); err != nil {
		t.Fatalf("RevokeAllForTenant returned error: %v", err)
	}

	expected := []string{
		fmt.Sprintf("apikey:%s", keyOne.KeyPrefix),
		fmt.Sprintf("apikey:id:%s", keyOne.ID.String()),
		fmt.Sprintf("apikey:%s", keyTwo.KeyPrefix),
		fmt.Sprintf("apikey:id:%s", keyTwo.ID.String()),
	}

	assertDeletedKeys(t, cache.deleted, expected)
}

func TestMarkExpiredKeysInvalidatesCache(t *testing.T) {
	tenantID := uuid.New()
	createdBy := uuid.New()
	repo := newMemoryAPIKeyRepo()

	expiredAt := time.Now().UTC().Add(-1 * time.Hour)

	keyActiveExpired, _, _ := apikey.NewAPIKey(tenantID, "expired-active", createdBy, []string{apikey.PermissionAll})
	keyActiveExpired.ExpiresAt = &expiredAt

	keySecondExpired, _, _ := apikey.NewAPIKey(tenantID, "expired-second", createdBy, []string{apikey.PermissionAll})
	keySecondExpired.ExpiresAt = &expiredAt

	keyValid, _, _ := apikey.NewAPIKey(tenantID, "valid", createdBy, []string{apikey.PermissionAll})

	_ = repo.Save(context.Background(), keyActiveExpired)
	_ = repo.Save(context.Background(), keySecondExpired)
	_ = repo.Save(context.Background(), keyValid)

	cache := &spyCache{}
	svc := NewService(apikey.NewService(repo), nil, nil, cache)

	count, err := svc.MarkExpiredKeys(context.Background(), 10)
	if err != nil {
		t.Fatalf("MarkExpiredKeys returned error: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected 2 expired keys processed, got %d", count)
	}

	expected := []string{
		fmt.Sprintf("apikey:%s", keyActiveExpired.KeyPrefix),
		fmt.Sprintf("apikey:id:%s", keyActiveExpired.ID.String()),
		fmt.Sprintf("apikey:%s", keySecondExpired.KeyPrefix),
		fmt.Sprintf("apikey:id:%s", keySecondExpired.ID.String()),
	}

	assertDeletedKeys(t, cache.deleted, expected)
}

func TestCreateAPIKeyRespectsQuota(t *testing.T) {
	tenantID := uuid.New()
	repo := newMemoryAPIKeyRepo()
	quotaRepo := &stubQuotaRepo{quota: &tenant.Quota{TenantID: tenantID, MaxAPIKeys: 1}}

	svc := NewService(apikey.NewService(repo), quotaRepo, nil, nil)
	ctx := withTenantClaims(context.Background(), tenantID)

	if _, _, err := svc.CreateAPIKey(ctx, tenantID, "first", uuid.New(), []string{apikey.PermissionAll}, 0); err != nil {
		t.Fatalf("expected first key to succeed, got %v", err)
	}

	_, _, err := svc.CreateAPIKey(ctx, tenantID, "second", uuid.New(), []string{apikey.PermissionAll}, 0)
	if !errors.Is(err, apikey.ErrQuotaExceeded) {
		t.Fatalf("expected quota exceeded error, got %v", err)
	}
}

func withTenantClaims(ctx context.Context, tenantID uuid.UUID) context.Context {
	claims := &authtypes.Claims{TenantID: tenantID.String(), Service: "test-service"}
	return authmw.WithClaims(ctx, claims)
}

type spyCache struct {
	deleted []string
}

func (c *spyCache) Get(_ context.Context, _ string, _ interface{}) error { return nil }

func (c *spyCache) Set(_ context.Context, _ string, _ interface{}, _ time.Duration) error { return nil }

func (c *spyCache) Delete(_ context.Context, key string) error {
	c.deleted = append(c.deleted, key)
	return nil
}

type memoryAPIKeyRepo struct {
	keys map[uuid.UUID]*apikey.APIKey
}

func newMemoryAPIKeyRepo() *memoryAPIKeyRepo {
	return &memoryAPIKeyRepo{keys: make(map[uuid.UUID]*apikey.APIKey)}
}

func (r *memoryAPIKeyRepo) Save(_ context.Context, key *apikey.APIKey) error {
	r.keys[key.ID] = key
	return nil
}

func (r *memoryAPIKeyRepo) Update(ctx context.Context, key *apikey.APIKey) error {
	return r.Save(ctx, key)
}

func (r *memoryAPIKeyRepo) FindByID(_ context.Context, id uuid.UUID) (*apikey.APIKey, error) {
	if key, ok := r.keys[id]; ok {
		return key, nil
	}
	return nil, apikey.ErrNotFound
}

func (r *memoryAPIKeyRepo) FindByTenantID(_ context.Context, tenantID uuid.UUID) ([]*apikey.APIKey, error) {
	var result []*apikey.APIKey
	for _, key := range r.keys {
		if key.TenantID == tenantID {
			result = append(result, key)
		}
	}
	return result, nil
}

func (r *memoryAPIKeyRepo) FindByKeyHash(_ context.Context, hash string) (*apikey.APIKey, error) {
	for _, key := range r.keys {
		if key.KeyHash == hash {
			return key, nil
		}
	}
	return nil, apikey.ErrNotFound
}

func (r *memoryAPIKeyRepo) FindByKeyPrefix(_ context.Context, prefix string) (*apikey.APIKey, error) {
	for _, key := range r.keys {
		if key.KeyPrefix == prefix {
			return key, nil
		}
	}
	return nil, apikey.ErrNotFound
}

func (r *memoryAPIKeyRepo) FindActiveByTenantID(_ context.Context, tenantID uuid.UUID) ([]*apikey.APIKey, error) {
	var result []*apikey.APIKey
	for _, key := range r.keys {
		if key.TenantID == tenantID && key.IsActive() {
			result = append(result, key)
		}
	}
	return result, nil
}

func (r *memoryAPIKeyRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.keys, id)
	return nil
}

func (r *memoryAPIKeyRepo) ExistsByName(_ context.Context, tenantID uuid.UUID, name string) (bool, error) {
	for _, key := range r.keys {
		if key.TenantID == tenantID && key.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (r *memoryAPIKeyRepo) CountByTenantID(_ context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	for _, key := range r.keys {
		if key.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (r *memoryAPIKeyRepo) CountActiveByTenantID(_ context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	for _, key := range r.keys {
		if key.TenantID == tenantID && key.IsActive() {
			count++
		}
	}
	return count, nil
}

func (r *memoryAPIKeyRepo) ListExpired(_ context.Context, limit int) ([]*apikey.APIKey, error) {
	var expired []*apikey.APIKey
	for _, key := range r.keys {
		if key.IsExpired() {
			expired = append(expired, key)
			if limit > 0 && len(expired) >= limit {
				break
			}
		}
	}
	return expired, nil
}

func (r *memoryAPIKeyRepo) RevokeAll(_ context.Context, tenantID uuid.UUID, revokedBy uuid.UUID) error {
	for _, key := range r.keys {
		if key.TenantID == tenantID {
			key.Revoke(revokedBy)
		}
	}
	return nil
}

type stubQuotaRepo struct {
	quota *tenant.Quota
}

func (s *stubQuotaRepo) GetQuota(_ context.Context, _ uuid.UUID) (*tenant.Quota, error) {
	return s.quota, nil
}

func assertDeletedKeys(t *testing.T, got []string, expected []string) {
	t.Helper()

	if len(got) != len(expected) {
		t.Fatalf("unexpected number of cache deletions: got %d want %d", len(got), len(expected))
	}

	seen := make(map[string]bool)
	for _, key := range got {
		seen[key] = true
	}

	for _, want := range expected {
		if !seen[want] {
			t.Fatalf("missing cache deletion for key %s", want)
		}
	}
}
