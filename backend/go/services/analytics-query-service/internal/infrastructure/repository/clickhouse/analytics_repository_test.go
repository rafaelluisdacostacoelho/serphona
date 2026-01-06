package clickhouse

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTenantEnforcer struct {
	mock.Mock
}

func (m *MockTenantEnforcer) EnforceTenant(ctx context.Context, tenantID string) error {
	args := m.Called(ctx, tenantID)
	return args.Error(0)
}

func TestGetCallMetrics_ValidTenantID(t *testing.T) {
	// Setup
	ctx := authmw.WithTenantID(context.Background(), "valid-tenant-id")
	mockEnforcer := new(MockTenantEnforcer)
	repo := AnalyticsRepository{enforcer: mockEnforcer}

	tenantID := "valid-tenant-id"
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	mockEnforcer.On("EnforceTenant", ctx, tenantID).Return(nil)

	// Replace MockDB with sqlmock
	mockDB, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer mockDB.Close()

	// Define expected query and arguments
	expectedQuery := `(?i)SELECT\s+count\(\)\s+as\s+total,\s+countIf\(status\s*=\s*'completed'\)\s+as\s+completed,\s+countIf\(status\s*=\s*'abandoned'\)\s+as\s+abandoned,\s+avg\(duration\)\s+as\s+avg_duration,\s+avg\(wait_time\)\s+as\s+avg_wait_time\s+FROM\s+analytics_events\s+WHERE\s+tenant_id\s*=\s*\?\s+AND\s+timestamp\s*>=\s*\?\s+AND\s+timestamp\s*<=\s*\?\s+AND\s+event_type\s*=\s*'call'`
	sqlMock.ExpectQuery(expectedQuery).
		WithArgs(tenantID, startTime, endTime).
		WillReturnRows(sqlmock.NewRows([]string{"total", "completed", "abandoned", "avg_duration", "avg_wait_time"}).
			AddRow(100, 80, 20, 30.5, 5.2))

	// Update repository to use sqlmock
	repo.db = mockDB

	// Execute
	metrics, err := repo.GetCallMetrics(ctx, tenantID, startTime, endTime)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, int64(100), metrics.Total)
	assert.Equal(t, int64(80), metrics.Completed)
	assert.Equal(t, int64(20), metrics.Abandoned)
	assert.Equal(t, 30.5, metrics.AvgDuration)
	assert.Equal(t, 5.2, metrics.AvgWaitTime)

	mockEnforcer.AssertExpectations(t)
	sqlMock.ExpectationsWereMet()
}

func TestGetCallMetrics_InvalidTenantID(t *testing.T) {
	// Setup
	ctx := authmw.WithTenantID(context.Background(), "invalid-tenant-id")
	mockEnforcer := new(MockTenantEnforcer)
	repo := AnalyticsRepository{enforcer: mockEnforcer}

	tenantID := "invalid-tenant-id"
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	mockEnforcer.On("EnforceTenant", ctx, tenantID).Return(fmt.Errorf("unauthorized"))

	// Execute
	metrics, err := repo.GetCallMetrics(ctx, tenantID, startTime, endTime)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, metrics)

	mockEnforcer.AssertExpectations(t)
}

func TestGetCallMetrics_MissingTenantContext(t *testing.T) {
	ctx := context.Background()
	mockEnforcer := new(MockTenantEnforcer)
	repo := AnalyticsRepository{enforcer: mockEnforcer}

	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer mockDB.Close()
	repo.db = mockDB

	metrics, err := repo.GetCallMetrics(ctx, "tenant-1", time.Now().Add(-time.Hour), time.Now())

	assert.ErrorIs(t, err, autherrors.ErrUnauthorized)
	assert.Nil(t, metrics)
	mockEnforcer.AssertNotCalled(t, "EnforceTenant", mock.Anything, mock.Anything)
}

func TestSearchEvents_EnforcesTenantAndReturnsResults(t *testing.T) {
	ctx := authmw.WithTenantID(context.Background(), "tenant-1")
	mockEnforcer := new(MockTenantEnforcer)
	repo := AnalyticsRepository{enforcer: mockEnforcer}

	start := time.Now().Add(-time.Hour)
	end := time.Now()
	filters := model.QueryFilters{TenantID: "tenant-1", StartTime: start, EndTime: end, Limit: 10, Offset: 0}

	mockEnforcer.On("EnforceTenant", ctx, filters.TenantID).Return(nil)

	mockDB, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer mockDB.Close()
	repo.db = mockDB

	mainQuery := `(?i)SELECT\s+event_id,\s+tenant_id,\s+session_id,\s+user_id,\s+agent_id,\s+event_type,\s+timestamp,\s+duration,\s+sentiment_score,\s+topics\s+FROM\s+analytics_events\s+WHERE\s+tenant_id\s*=\s*\?` // fuzzed match
	sqlMock.ExpectQuery(mainQuery).
		WithArgs(filters.TenantID, filters.StartTime, filters.EndTime, filters.Limit, filters.Offset).
		WillReturnRows(sqlmock.NewRows([]string{"event_id", "tenant_id", "session_id", "user_id", "agent_id", "event_type", "timestamp", "duration", "sentiment_score", "topics"}).
			AddRow("evt-1", filters.TenantID, "sess-1", "user-1", "agent-1", "call", time.Now(), int64(10), float64(0.5), "[]"))

	countQuery := `(?i)SELECT\s+count\(\)\s+FROM\s+analytics_events\s+WHERE\s+tenant_id\s*=\s*\?\s+AND\s+timestamp\s*>=\s*\?\s+AND\s+timestamp\s*<=\s*\?`
	sqlMock.ExpectQuery(countQuery).
		WithArgs(filters.TenantID, filters.StartTime, filters.EndTime).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	events, total, err := repo.SearchEvents(ctx, filters)

	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, int64(1), total)
	mockEnforcer.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestSearchEvents_MissingTenantContext(t *testing.T) {
	repo := AnalyticsRepository{}
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer mockDB.Close()
	repo.db = mockDB

	filters := model.QueryFilters{TenantID: "tenant-1", StartTime: time.Now().Add(-time.Hour), EndTime: time.Now()}

	events, total, err := repo.SearchEvents(context.Background(), filters)

	assert.ErrorIs(t, err, autherrors.ErrUnauthorized)
	assert.Nil(t, events)
	assert.Equal(t, int64(0), total)
}

func TestSearchEvents_TenantMismatchFromEnforcer(t *testing.T) {
	ctx := authmw.WithTenantID(context.Background(), "tenant-1")
	mockEnforcer := new(MockTenantEnforcer)
	repo := AnalyticsRepository{enforcer: mockEnforcer}

	filters := model.QueryFilters{TenantID: "tenant-1", StartTime: time.Now().Add(-time.Hour), EndTime: time.Now()}
	mockEnforcer.On("EnforceTenant", ctx, filters.TenantID).Return(fmt.Errorf("deny"))

	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer mockDB.Close()
	repo.db = mockDB

	events, total, err := repo.SearchEvents(ctx, filters)

	assert.Error(t, err)
	assert.Nil(t, events)
	assert.Equal(t, int64(0), total)
	mockEnforcer.AssertExpectations(t)
}
