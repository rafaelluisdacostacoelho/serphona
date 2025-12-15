package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap/zaptest"

	"tenant-manager/internal/application/tenant"
	domain "tenant-manager/internal/domain/tenant"
	apperrors "tenant-manager/pkg/errors"
)

// In-memory tenant repository for handler tests.
type tenantRepoStub struct {
	items map[uuid.UUID]*domain.Tenant
}

func newTenantRepoStub() *tenantRepoStub {
	return &tenantRepoStub{items: make(map[uuid.UUID]*domain.Tenant)}
}

func (r *tenantRepoStub) Create(_ context.Context, t *domain.Tenant) error {
	r.items[t.ID] = t
	return nil
}
func (r *tenantRepoStub) GetByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	if t, ok := r.items[id]; ok {
		return t, nil
	}
	return nil, apperrors.NewNotFoundError("not found")
}
func (r *tenantRepoStub) GetBySlug(_ context.Context, slug string) (*domain.Tenant, error) {
	for _, t := range r.items {
		if t.Slug == slug {
			return t, nil
		}
	}
	return nil, apperrors.NewNotFoundError("not found")
}
func (r *tenantRepoStub) GetByEmail(_ context.Context, email string) (*domain.Tenant, error) {
	for _, t := range r.items {
		if t.Email == email {
			return t, nil
		}
	}
	return nil, apperrors.NewNotFoundError("not found")
}
func (r *tenantRepoStub) Update(_ context.Context, t *domain.Tenant) error {
	r.items[t.ID] = t
	return nil
}
func (r *tenantRepoStub) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}
func (r *tenantRepoStub) List(_ context.Context, filter domain.ListFilter) (*domain.ListResult, error) {
	var tenants []*domain.Tenant
	for _, t := range r.items {
		tenants = append(tenants, t)
	}
	total := len(tenants)
	return &domain.ListResult{
		Tenants:    tenants,
		Total:      int64(total),
		PageSize:   filter.PageSize,
		PageNumber: filter.PageNumber,
		TotalPages: 1,
	}, nil
}
func (r *tenantRepoStub) UpdateSettings(_ context.Context, id uuid.UUID, settings domain.Settings) error {
	if t, ok := r.items[id]; ok {
		t.Settings = settings
		return nil
	}
	return apperrors.NewNotFoundError("not found")
}
func (r *tenantRepoStub) GetQuota(_ context.Context, tenantID uuid.UUID) (*domain.Quota, error) {
	return nil, apperrors.NewNotFoundError("not found")
}
func (r *tenantRepoStub) UpdateQuota(_ context.Context, quota *domain.Quota) error { return nil }
func (r *tenantRepoStub) IncrementUsage(_ context.Context, tenantID uuid.UUID, calls, minutes int) error {
	return nil
}
func (r *tenantRepoStub) ExistsBySlug(_ context.Context, slug string) (bool, error) {
	for _, t := range r.items {
		if t.Slug == slug {
			return true, nil
		}
	}
	return false, nil
}
func (r *tenantRepoStub) ExistsByEmail(_ context.Context, email string) (bool, error) {
	for _, t := range r.items {
		if t.Email == email {
			return true, nil
		}
	}
	return false, nil
}

type noopCache struct{}

func (noopCache) Get(_ context.Context, _ string) (*domain.Tenant, error)              { return nil, nil }
func (noopCache) Set(_ context.Context, _ string, _ *domain.Tenant) error              { return nil }
func (noopCache) Delete(_ context.Context, _ string) error                             { return nil }
func (noopCache) GetSettings(_ context.Context, _ uuid.UUID) (*domain.Settings, error) { return nil, nil }
func (noopCache) SetSettings(_ context.Context, _ uuid.UUID, _ *domain.Settings) error { return nil }
func (noopCache) Invalidate(_ context.Context, _ uuid.UUID) error                      { return nil }

type noopPublisher struct{}

func (noopPublisher) PublishCreated(_ context.Context, _ *domain.Tenant) error                     { return nil }
func (noopPublisher) PublishUpdated(_ context.Context, _ *domain.Tenant) error                     { return nil }
func (noopPublisher) PublishDeleted(_ context.Context, _ uuid.UUID) error                          { return nil }
func (noopPublisher) PublishActivated(_ context.Context, _ *domain.Tenant) error                   { return nil }
func (noopPublisher) PublishSuspended(_ context.Context, _ *domain.Tenant) error                   { return nil }
func (noopPublisher) PublishSettingsUpdated(_ context.Context, _ uuid.UUID, _ *domain.Settings) error { return nil }

func TestGinTenantHandlerCreateAndList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newTenantRepoStub()
	svc := tenant.NewService(repo, nil, noopCache{}, noopPublisher{}, zaptest.NewLogger(t))
	handler := NewGinTenantHandler(svc, zaptest.NewLogger(t))

	// Create
	body := `{"name":"Acme","email":"acme@example.com","plan":"starter"}`
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewBufferString(body))
	handler.Create(c)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	// List
	rec2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(rec2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tenants?page=1&page_size=10", nil)
	handler.List(c2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 on list, got %d", rec2.Code)
	}
	var listResp ListTenantsResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to parse list response: %v", err)
	}
	if len(listResp.Tenants) != 1 {
		t.Fatalf("expected 1 tenant, got %d", len(listResp.Tenants))
	}
}
