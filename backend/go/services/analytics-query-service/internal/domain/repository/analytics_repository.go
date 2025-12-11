package repository

import (
	"context"
	"time"

	"github.com/serphona/serphona/backend/go/services/analytics-query-service/internal/domain/model"
)

// AnalyticsRepository defines methods for querying analytics data
type AnalyticsRepository interface {
	// Overview metrics
	GetOverviewMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.OverviewMetrics, error)

	// Call metrics
	GetCallMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.CallMetrics, error)

	// Sentiment metrics
	GetSentimentMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.SentimentMetrics, error)

	// Topic metrics
	GetTopicMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time, limit int) ([]model.TopicMetric, error)

	// Agent metrics
	GetAgentMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) ([]model.AgentMetric, error)

	// Time series
	GetCallTimeSeries(ctx context.Context, tenantID string, startTime, endTime time.Time, granularity string) (*model.TimeSeriesData, error)
	GetSentimentTimeSeries(ctx context.Context, tenantID string, startTime, endTime time.Time, granularity string) (*model.TimeSeriesData, error)

	// Aggregations
	GetAggregations(ctx context.Context, tenantID string, startTime, endTime time.Time, granularity string) ([]model.AggregationResult, error)

	// Search events
	SearchEvents(ctx context.Context, filters model.QueryFilters) ([]model.AnalyticsEvent, int64, error)
}
