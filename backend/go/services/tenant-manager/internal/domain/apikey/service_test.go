package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

// In-memory repository stub for tests.
type testRepo struct {
	keys map[uuid.UUID]*APIKey
}

func newTestRepo() *testRepo {
	return &testRepo{keys: make(map[uuid.UUID]*APIKey)}
}

func (r *testRepo) Save(_ context.Context, key *APIKey) error {
	r.keys[key.ID] = key
	return nil
}

func (r *testRepo) Update(_ context.Context, key *APIKey) error {
	if _, ok := r.keys[key.ID]; !ok {
		return ErrNotFound
	}
	r.keys[key.ID] = key
	return nil
}

func (r *testRepo) FindByID(_ context.Context, id uuid.UUID) (*APIKey, error) {
	key, ok := r.keys[id]
	if !ok {
		return nil, ErrNotFound
	}
	return key, nil
}

func (r *testRepo) FindByTenantID(_ context.Context, tenantID uuid.UUID) ([]*APIKey, error) {
	var result []*APIKey
	for _, k := range r.keys {
		if k.TenantID == tenantID {
			result = append(result, k)
		}
	}
	return result, nil
}

func (r *testRepo) FindByKeyHash(_ context.Context, keyHash string) (*APIKey, error) {
	for _, k := range r.keys {
		if k.KeyHash == keyHash {
			return k, nil
		}
	}
	return nil, ErrNotFound
}

func (r *testRepo) FindByKeyPrefix(_ context.Context, prefix string) (*APIKey, error) {
	for _, k := range r.keys {
		if k.KeyPrefix == prefix {
			return k, nil
		}
	}
	return nil, ErrNotFound
}

func (r *testRepo) FindActiveByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*APIKey, error) {
	var result []*APIKey
	for _, k := range r.keys {
		if k.TenantID == tenantID && k.RevokedAt == nil && (k.ExpiresAt == nil || time.Now().UTC().Before(*k.ExpiresAt)) {
			result = append(result, k)
		}
	}
	return result, nil
}

func (r *testRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.keys, id)
	return nil
}

func (r *testRepo) ExistsByName(_ context.Context, tenantID uuid.UUID, name string) (bool, error) {
	for _, k := range r.keys {
		if k.TenantID == tenantID && k.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (r *testRepo) CountByTenantID(_ context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	for _, k := range r.keys {
		if k.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (r *testRepo) CountActiveByTenantID(_ context.Context, tenantID uuid.UUID) (int64, error) {
	var count int64
	for _, k := range r.keys {
		if k.TenantID == tenantID && k.RevokedAt == nil && (k.ExpiresAt == nil || time.Now().UTC().Before(*k.ExpiresAt)) {
			count++
		}
	}
	return count, nil
}

func (r *testRepo) ListExpired(_ context.Context, limit int) ([]*APIKey, error) {
	var result []*APIKey
	now := time.Now().UTC()
	for _, k := range r.keys {
		if k.ExpiresAt != nil && now.After(*k.ExpiresAt) && k.RevokedAt == nil {
			result = append(result, k)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (r *testRepo) RevokeAll(_ context.Context, tenantID uuid.UUID, _ uuid.UUID) error {
	now := time.Now().UTC()
	for _, k := range r.keys {
		if k.TenantID == tenantID && k.RevokedAt == nil {
			k.RevokedAt = &now
		}
	}
	return nil
}

func TestAPIKeyServiceCreateAndAuthenticate(t *testing.T) {
	repo := newTestRepo()
	svc := NewService(repo)

	tenantID := uuid.New()
	userID := uuid.New()
	ctx := authmw.WithTenantID(context.Background(), tenantID.String())
	key, raw, err := svc.Create(ctx, tenantID, "my-key", userID, []string{PermissionAll}, 0)
	if err != nil {
		t.Fatalf("unexpected error creating key: %v", err)
	}
	if raw == "" || len(raw) < 20 || raw[:3] != "sk_" {
		t.Fatalf("raw key not returned or invalid format: %q", raw)
	}
	if key.KeyHash != "" {
		t.Fatalf("sanitized key should not expose hash")
	}

	// Authenticate with the raw key
	authKey, err := svc.Authenticate(ctx, raw, "127.0.0.1")
	if err != nil {
		t.Fatalf("authenticate failed: %v", err)
	}
	if authKey.ID != key.ID {
		t.Fatalf("expected key ID %s, got %s", key.ID, authKey.ID)
	}
}

func TestAPIKeyServiceCreateValidation(t *testing.T) {
	repo := newTestRepo()
	svc := NewService(repo)
	tenantID := uuid.New()
	userID := uuid.New()

	ctx := authmw.WithTenantID(context.Background(), tenantID.String())

	_, _, err := svc.Create(ctx, tenantID, "", userID, []string{PermissionAll}, 0)
	if err == nil {
		t.Fatalf("expected error for empty name")
	}

	_, _, err = svc.Create(ctx, tenantID, "key", userID, []string{"invalid:permission"}, 0)
	if err == nil {
		t.Fatalf("expected error for invalid permission")
	}
}
