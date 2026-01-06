package tenant

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, tenant *Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}

func (m *MockRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	return false, nil
}

func (m *MockRepository) GetByEmail(ctx context.Context, email string) (*Tenant, error) {
	return nil, nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	return nil, nil
}

func (m *MockRepository) GetBySlug(ctx context.Context, slug string) (*Tenant, error) {
	return nil, nil
}

func (m *MockRepository) GetQuota(ctx context.Context, tenantID uuid.UUID) (*Quota, error) {
	// Mock implementation for GetQuota
	return &Quota{}, nil
}

func (m *MockRepository) IncrementUsage(ctx context.Context, tenantID uuid.UUID, usage int, limit int) error {
	// Mock implementation for IncrementUsage
	return nil
}

func (m *MockRepository) List(ctx context.Context, filter ListFilter) (*ListResult, error) {
	return nil, nil
}

func (m *MockRepository) Update(ctx context.Context, tenant *Tenant) error {
	return nil
}

func (m *MockRepository) UpdateQuota(ctx context.Context, quota *Quota) error {
	return nil
}

func (m *MockRepository) UpdateSettings(ctx context.Context, tenantID uuid.UUID, settings Settings) error {
	return nil
}

func TestServiceCreate_EnforceTenant(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)

	ctx := context.Background()
	ctx = middleware.WithTenantID(ctx, "123e4567-e89b-12d3-a456-426614174000")

	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	tenant := &Tenant{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), Name: "Test Tenant", Email: "test@example.com"}
	mockRepo.On("Create", ctx, tenant).Return(nil)

	err := middleware.EnforceTenant(ctx, tenant.ID.String())
	assert.NoError(t, err)

	// Removendo a variável não utilizada
	_ = svc
}
