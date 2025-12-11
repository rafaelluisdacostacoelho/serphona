package usecase

import (
	"context"
	"time"

	"github.com/serphona/serphona/backend/go/services/analytics-query-service/internal/domain/model"
	"github.com/serphona/serphona/backend/go/services/analytics-query-service/internal/domain/repository"
)

type AnalyticsService struct {
	repo repository.AnalyticsRepository
}

func NewAnalyticsService(repo repository.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

func (s *AnalyticsService) GetOverviewMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.OverviewMetrics, error) {
	return s.repo.GetOverviewMetrics(ctx, tenantID, startTime, endTime)
}

func (s *AnalyticsService) GetCallMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.CallMetrics, error) {
	return s.repo.GetCallMetrics(ctx, tenantID, startTime, endTime)
}

func (s *AnalyticsService) GetSentimentMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.SentimentMetrics, error) {
	return s.repo.GetSentimentMetrics(ctx, tenantID, startTime, endTime)
}

func (s *AnalyticsService) GetTopicMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time, limit int) ([]model.TopicMetric, error) {
	if limit == 0 {
		limit = 10
	}
	return s.repo.GetTopicMetrics(ctx, tenantID, startTime, endTime, limit)
}

func (s *AnalyticsService) GetAgentMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) ([]model.AgentMetric, error) {
	return s.repo.GetAgentMetrics(ctx, tenantID, startTime, endTime)
}

func (s *AnalyticsService) GetCallTimeSeries(ctx context.Context, tenantID string, startTime, endTime time.Time, granularity string) (*model.TimeSeriesData, error) {
	return s.repo.GetCallTimeSeries(ctx, tenantID, startTime, endTime, granularity)
}

func (s *AnalyticsService) GetSentimentTimeSeries(ctx context.Context, tenantID string, startTime, endTime time.Time, granularity string) (*model.TimeSeriesData, error) {
	return s.repo.GetSentimentTimeSeries(ctx, tenantID, startTime, endTime, granularity)
}

func (s *AnalyticsService) GetAggregations(ctx context.Context, tenantID string, startTime, endTime time.Time, granularity string) ([]model.AggregationResult, error) {
	return s.repo.GetAggregations(ctx, tenantID, startTime, endTime, granularity)
}

func (s *AnalyticsService) SearchEvents(ctx context.Context, filters model.QueryFilters) ([]model.AnalyticsEvent, int64, error) {
	return s.repo.SearchEvents(ctx, filters)
}
