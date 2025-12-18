//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	httpHandler "tenant-manager/internal/adapter/http/handler"
	appTenant "tenant-manager/internal/application/tenant"
	"tenant-manager/internal/domain/tenant"
)

func TestTenantCreateIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		seed     func(*fakeTenantRepo)
		payload  map[string]any
		wantCode int
	}{
		{
			name: "creates tenant and returns slug",
			seed: nil,
			payload: map[string]any{
				"name":  "Acme Corp",
				"email": "owner@acme.test",
				"plan":  "starter",
			},
			wantCode: http.StatusCreated,
		},
		{
			name: "conflict on duplicate email",
			seed: func(r *fakeTenantRepo) {
				existing := tenant.NewTenant("Existing", "owner@acme.test", tenant.PlanStarter)
				existing.Slug = "existing"
				_ = r.Create(nil, existing)
			},
			payload: map[string]any{
				"name":  "Acme Corp",
				"email": "owner@acme.test",
				"plan":  "starter",
			},
			wantCode: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeTenantRepo()
			cache := newFakeTenantCache()
			publisher := &noopTenantPublisher{}
			apiRepo := &noopAPIKeyRepo{}
			svc := appTenant.NewService(repo, apiRepo, cache, publisher, zap.NewNop())
			handler := httpHandler.NewGinTenantHandler(svc, zap.NewNop())

			if tt.seed != nil {
				tt.seed(repo)
			}

			router := gin.New()
			router.POST("/api/v1/tenants", handler.Create)

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("unexpected status: got %d want %d body=%s", w.Code, tt.wantCode, w.Body.String())
			}

			if w.Code == http.StatusCreated {
				var resp httpHandler.TenantResponse
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Email != tt.payload["email"] {
					t.Fatalf("email mismatch: got %s", resp.Email)
				}
				if resp.Slug == "" {
					t.Fatalf("expected non-empty slug")
				}
			}
		})
	}
}

// fakeTenantRepo is an in-memory repo implementing tenant.Repository for integration tests.
type fakeTenantRepo struct {
	mu     sync.Mutex
	byID   map[uuid.UUID]*tenant.Tenant
	bySlug map[string]uuid.UUID
	byMail map[string]uuid.UUID
}

func newFakeTenantRepo() *fakeTenantRepo {
	return &fakeTenantRepo{
		byID:   make(map[uuid.UUID]*tenant.Tenant),
		bySlug: make(map[string]uuid.UUID),
		byMail: make(map[string]uuid.UUID),
	}
}

func (f *fakeTenantRepo) Create(_ context.Context, tnt *tenant.Tenant) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[tnt.ID] = tnt
	if tnt.Slug != "" {
		f.bySlug[tnt.Slug] = tnt.ID
	}
	if tnt.Email != "" {
		f.byMail[tnt.Email] = tnt.ID
	}
	return nil
}

func (f *fakeTenantRepo) GetByID(_ context.Context, id uuid.UUID) (*tenant.Tenant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t, ok := f.byID[id]; ok {
		return t, nil
	}
	return nil, tenant.ErrNotFound
}

func (f *fakeTenantRepo) GetBySlug(_ context.Context, slug string) (*tenant.Tenant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id, ok := f.bySlug[slug]; ok {
		return f.byID[id], nil
	}
	return nil, tenant.ErrNotFound
}

func (f *fakeTenantRepo) GetByEmail(_ context.Context, email string) (*tenant.Tenant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id, ok := f.byMail[email]; ok {
		return f.byID[id], nil
	}
	return nil, tenant.ErrNotFound
}

func (f *fakeTenantRepo) Update(ctx context.Context, tnt *tenant.Tenant) error {
	return f.Create(ctx, tnt)
}

func (f *fakeTenantRepo) Delete(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byID, id)
	for slug, sid := range f.bySlug {
		if sid == id {
			delete(f.bySlug, slug)
		}
	}
	for mail, mid := range f.byMail {
		if mid == id {
			delete(f.byMail, mail)
		}
	}
	return nil
}

func (f *fakeTenantRepo) List(_ context.Context, _ tenant.ListFilter) (*tenant.ListResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	tenants := make([]*tenant.Tenant, 0, len(f.byID))
	for _, tnt := range f.byID {
		tenants = append(tenants, tnt)
	}
	return &tenant.ListResult{
		Tenants:    tenants,
		Total:      int64(len(tenants)),
		PageSize:   len(tenants),
		PageNumber: 1,
		TotalPages: 1,
	}, nil
}

func (f *fakeTenantRepo) UpdateSettings(_ context.Context, _ uuid.UUID, _ tenant.Settings) error {
	return nil
}
func (f *fakeTenantRepo) GetQuota(_ context.Context, _ uuid.UUID) (*tenant.Quota, error) {
	return &tenant.Quota{}, nil
}
func (f *fakeTenantRepo) UpdateQuota(_ context.Context, _ *tenant.Quota) error { return nil }
func (f *fakeTenantRepo) IncrementUsage(_ context.Context, _ uuid.UUID, _ int, _ int) error {
	return nil
}

func (f *fakeTenantRepo) ExistsBySlug(_ context.Context, slug string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.bySlug[slug]
	return ok, nil
}

func (f *fakeTenantRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.byMail[email]
	return ok, nil
}

// fakeTenantCache is an in-memory cache.
type fakeTenantCache struct {
	mu    sync.Mutex
	store map[string]*tenant.Tenant
}

func newFakeTenantCache() *fakeTenantCache {
	return &fakeTenantCache{store: make(map[string]*tenant.Tenant)}
}

func (c *fakeTenantCache) Get(_ context.Context, key string) (*tenant.Tenant, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.store[key], nil
}

func (c *fakeTenantCache) Set(_ context.Context, key string, tnt *tenant.Tenant) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = tnt
	return nil
}

func (c *fakeTenantCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
	return nil
}

func (c *fakeTenantCache) GetSettings(_ context.Context, _ uuid.UUID) (*tenant.Settings, error) {
	return nil, nil
}
func (c *fakeTenantCache) SetSettings(_ context.Context, _ uuid.UUID, _ *tenant.Settings) error {
	return nil
}
func (c *fakeTenantCache) Invalidate(_ context.Context, _ uuid.UUID) error { return nil }

// noopTenantPublisher publishes nothing; used in integration tests.
type noopTenantPublisher struct{}

func (n *noopTenantPublisher) PublishCreated(_ context.Context, _ *tenant.Tenant) error {
	return nil
}
func (n *noopTenantPublisher) PublishUpdated(_ context.Context, _ *tenant.Tenant) error {
	return nil
}
func (n *noopTenantPublisher) PublishDeleted(_ context.Context, _ uuid.UUID) error { return nil }
func (n *noopTenantPublisher) PublishActivated(_ context.Context, _ *tenant.Tenant) error {
	return nil
}
func (n *noopTenantPublisher) PublishSuspended(_ context.Context, _ *tenant.Tenant) error { return nil }
func (n *noopTenantPublisher) PublishSettingsUpdated(_ context.Context, _ uuid.UUID, _ *tenant.Settings) error {
	return nil
}

// noopAPIKeyRepo fakes API key operations.
type noopAPIKeyRepo struct{}

func (n *noopAPIKeyRepo) GenerateAPIKey(_ context.Context, tenantID uuid.UUID) (string, error) {
	return "api-" + tenantID.String(), nil
}

func (n *noopAPIKeyRepo) ValidateAPIKey(_ context.Context, _ string) (*uuid.UUID, error) {
	return nil, nil
}
