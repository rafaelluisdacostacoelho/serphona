package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/repository"
	"gorm.io/gorm"
)

// toolExecutionRepositoryImpl implements the ToolExecutionRepository interface
type toolExecutionRepositoryImpl struct {
	db *gorm.DB
}

// NewToolExecutionRepository creates a new instance of ToolExecutionRepository
func NewToolExecutionRepository(db *gorm.DB) repository.ToolExecutionRepository {
	return &toolExecutionRepositoryImpl{db: db}
}

// Create creates a new tool execution record
func (r *toolExecutionRepositoryImpl) Create(ctx context.Context, execution *entity.ToolExecution) error {
	return r.db.WithContext(ctx).Create(execution).Error
}

// FindByID finds a tool execution by ID
func (r *toolExecutionRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.ToolExecution, error) {
	var execution entity.ToolExecution
	err := r.db.WithContext(ctx).
		Preload("Tool").
		Where("id = ?", id).
		First(&execution).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tool execution not found: %w", err)
		}
		return nil, err
	}
	return &execution, nil
}

// FindByTenant finds all executions for a tenant with filters
func (r *toolExecutionRepositoryImpl) FindByTenant(ctx context.Context, tenantID uuid.UUID, filters repository.ExecutionFilters) ([]*entity.ToolExecution, int64, error) {
	var executions []*entity.ToolExecution
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.ToolExecution{}).Where("tenant_id = ?", tenantID)

	// Apply filters
	if filters.ToolID != nil {
		query = query.Where("tool_id = ?", *filters.ToolID)
	}

	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	if filters.FromDate != nil {
		query = query.Where("executed_at >= ?", *filters.FromDate)
	}

	if filters.ToDate != nil {
		query = query.Where("executed_at <= ?", *filters.ToDate)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}

	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	// Order by executed_at desc
	query = query.Preload("Tool").Order("executed_at DESC")

	if err := query.Find(&executions).Error; err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

// FindByTool finds all executions for a specific tool
func (r *toolExecutionRepositoryImpl) FindByTool(ctx context.Context, toolID uuid.UUID, filters repository.ExecutionFilters) ([]*entity.ToolExecution, int64, error) {
	var executions []*entity.ToolExecution
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.ToolExecution{}).Where("tool_id = ?", toolID)

	// Apply filters
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	if filters.FromDate != nil {
		query = query.Where("executed_at >= ?", *filters.FromDate)
	}

	if filters.ToDate != nil {
		query = query.Where("executed_at <= ?", *filters.ToDate)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}

	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	// Order by executed_at desc
	query = query.Preload("Tool").Order("executed_at DESC")

	if err := query.Find(&executions).Error; err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

// Update updates an existing execution
func (r *toolExecutionRepositoryImpl) Update(ctx context.Context, execution *entity.ToolExecution) error {
	return r.db.WithContext(ctx).Save(execution).Error
}

// GetStats gets execution statistics
func (r *toolExecutionRepositoryImpl) GetStats(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*repository.ExecutionStats, error) {
	var stats repository.ExecutionStats

	// Base query
	query := r.db.WithContext(ctx).
		Model(&entity.ToolExecution{}).
		Where("tenant_id = ? AND executed_at BETWEEN ? AND ?", tenantID, from, to)

	// Total executions
	if err := query.Count(&stats.TotalExecutions).Error; err != nil {
		return nil, err
	}

	// Count by status
	var statusCounts []struct {
		Status string
		Count  int64
	}

	if err := query.
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		return nil, err
	}

	for _, sc := range statusCounts {
		switch sc.Status {
		case entity.StatusSuccess:
			stats.SuccessfulCount = sc.Count
		case entity.StatusError:
			stats.FailedCount = sc.Count
		case entity.StatusTimeout:
			stats.TimeoutCount = sc.Count
		case entity.StatusRateLimited:
			stats.RateLimitedCount = sc.Count
		}
	}

	// Total credits and average latency
	var aggregates struct {
		TotalCredits int64
		AvgLatency   float64
	}

	if err := query.
		Select("COALESCE(SUM(credits_consumed), 0) as total_credits, COALESCE(AVG(latency_ms), 0) as avg_latency").
		Scan(&aggregates).Error; err != nil {
		return nil, err
	}

	stats.TotalCredits = aggregates.TotalCredits
	stats.AverageLatencyMS = aggregates.AvgLatency

	// Calculate success rate
	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessfulCount) / float64(stats.TotalExecutions) * 100
	}

	return &stats, nil
}

// GetCostBreakdown gets cost breakdown by tool
func (r *toolExecutionRepositoryImpl) GetCostBreakdown(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]*repository.CostBreakdown, error) {
	var breakdowns []*repository.CostBreakdown

	query := `
		SELECT 
			te.tool_id,
			t.name as tool_name,
			COUNT(*) as execution_count,
			COALESCE(SUM(te.credits_consumed), 0) as total_credits,
			COALESCE(SUM(CASE WHEN te.status = ? THEN 1 ELSE 0 END), 0) as success_count,
			COALESCE(SUM(CASE WHEN te.status IN (?, ?) THEN 1 ELSE 0 END), 0) as failed_count,
			COALESCE(AVG(te.latency_ms), 0) as average_latency
		FROM tool_executions te
		JOIN tools t ON te.tool_id = t.id
		WHERE te.tenant_id = ? AND te.executed_at BETWEEN ? AND ?
		GROUP BY te.tool_id, t.name
		ORDER BY total_credits DESC
	`

	err := r.db.WithContext(ctx).Raw(
		query,
		entity.StatusSuccess,
		entity.StatusError,
		entity.StatusTimeout,
		tenantID,
		from,
		to,
	).Scan(&breakdowns).Error

	if err != nil {
		return nil, err
	}

	return breakdowns, nil
}
