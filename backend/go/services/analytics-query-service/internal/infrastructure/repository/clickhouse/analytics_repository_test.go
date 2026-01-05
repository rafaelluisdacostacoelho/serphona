package clickhouse

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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
	ctx := context.Background()
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
	ctx := context.Background()
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
