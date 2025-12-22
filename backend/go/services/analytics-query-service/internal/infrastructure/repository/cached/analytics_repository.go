package cached

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/domain/model"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/domain/repository"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/infrastructure/cache"
)

// CachedAnalyticsRepository wraps a repository with Redis caching
type CachedAnalyticsRepository struct {
	repo  repository.AnalyticsRepository
	cache *cache.RedisCache
}

func NewCachedAnalyticsRepository(repo repository.AnalyticsRepository, redisClient *redis.Client, ttl time.Duration) *CachedAnalyticsRepository {
	return &CachedAnalyticsRepository{
		repo:  repo,
		cache: cache.NewRedisCache(redisClient, ttl),
	}
}

func (r *CachedAnalyticsRepository) GetOverviewMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.OverviewMetrics, error) {
	key := cache.GenerateKey("overview", tenantID, startTime.Unix(), endTime.Unix())

	var metrics model.OverviewMetrics
	err := r.cache.Get(ctx, key, &metrics)
	if err == nil {
		log.Printf("✅ Cache hit: %s", key)
		return &metrics, nil
	}

	log.Printf("⚠️  Cache miss: %s", key)
	result, err := r.repo.GetOverviewMetrics(ctx, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	_ = r.cache.Set(ctx, key, result)
	return result, nil
}

func (r *CachedAnalyticsRepository) GetCallMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.CallMetrics, error) {
	key := cache.GenerateKey("calls", tenantID, startTime.Unix(), endTime.Unix())

	var metrics model.CallMetrics
	err := r.cache.Get(ctx, key, &metrics)
	if err == nil {
		log.Printf("✅ Cache hit: %s", key)
		return &metrics, nil
	}

	log.Printf("⚠️  Cache miss: %s", key)
	result, err := r.repo.GetCallMetrics(ctx, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	_ = r.cache.Set(ctx, key, result)
	return result, nil
}

func (r *CachedAnalyticsRepository) GetSentimentMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.SentimentMetrics, error) {
	key := cache.GenerateKey("sentiment", tenantID, startTime.Unix(), endTime.Unix())

	var metrics model.SentimentMetrics
	err := r.cache.Get(ctx, key, &metrics)
	if err == nil {
		log.Printf("✅ Cache hit: %s", key)
		return &metrics, nil
	}

	log.Printf("⚠️  Cache miss: %s", key)
	result, err := r.repo.GetSentimentMetrics(ctx, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	_ = r.cache.Set(ctx, key, result)
	return result, nil
}

func (r *CachedAnalyticsRepository) GetTopicMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time, limit int) ([]model.TopicMetric, error) {
	key := cache.GenerateKey("topics", tenantID, startTime.Unix(), endTime.Unix(), limit)

	var metrics []model.TopicMetric
	err := r.cache.Get(ctx, key, &metrics)
	if err == nil {
		log.Printf("✅ Cache hit: %s", key)
		return metrics, nil
	}

	log.Printf("⚠️  Cache miss: %s", key)
	result, err := r.repo.GetTopicMetrics(ctx, tenantID, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}

	_ = r.cache.Set(ctx, key, result)
	return result, nil
}

func (r *CachedAnalyticsRepository) GetAgentMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) ([]model.AgentMetric, error) {
	key := cache.GenerateKey("agents", tenantID, startTime.Unix(), endTime.Unix())

	var metrics []model.AgentMetric
	err := r.cache.Get(ctx, key, &metrics)
	if err == nil {
		log.Printf("✅ Cache hit: %s", key)
		return metrics, nil
	}

	log.Printf("⚠️  Cache miss: %s", key)
	result, err := r.repo.GetAgentMetrics(ctx, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	_ = r.cache.Set(ctx, key, result)
	return result, nil
}

func (r *CachedAnalyticsRepository) GetCallTimeSeries(ctx context.Context, tenantID string, startTime, endTime time.Time, granularity string) (*model.TimeSeriesData, error) {
	key := cache.GenerateKey("timeseries:calls", tenantID, startTime.Unix(), endTime.Unix(), granularity)

	var data model.TimeSeriesData
	err := r.cache.Get(ctx, key, &data)
	if err == nil {
		log.Printf("✅ Cache hit: %s", key)
		return &data, nil
	}

	log.Printf("⚠️  Cache miss: %s", key)
	result, err := r.repo.GetCallTimeSeries(ctx, tenantID, startTime, endTime, granularity)
	if err != nil {
		return nil, err
	}

	_ = r.cache.Set(ctx, key, result)
	return result, nil
}

func (r *CachedAnalyticsRepository) GetSentimentTimeSeries(ctx context.Context, tenantID string, startTime, endTime time.Time, granularity string) (*model.TimeSeriesData, error) {
	key := cache.GenerateKey("timeseries:sentiment", tenantID, startTime.Unix(), endTime.Unix(), granularity)

	var data model.TimeSeriesData
	err := r.cache.Get(ctx, key, &data)
	if err == nil {
		log.Printf("✅ Cache hit: %s", key)
		return &data, nil
	}

	log.Printf("⚠️  Cache miss: %s", key)
	result, err := r.repo.GetSentimentTimeSeries(ctx, tenantID, startTime, endTime, granularity)
	if err != nil {
		return nil, err
	}

	_ = r.cache.Set(ctx, key, result)
	return result, nil
}

func (r *CachedAnalyticsRepository) GetAggregations(ctx context.Context, tenantID string, startTime, endTime time.Time, granularity string) ([]model.AggregationResult, error) {
	key := cache.GenerateKey("aggregations", tenantID, startTime.Unix(), endTime.Unix(), granularity)

	var results []model.AggregationResult
	err := r.cache.Get(ctx, key, &results)
	if err == nil {
		log.Printf("✅ Cache hit: %s", key)
		return results, nil
	}

	log.Printf("⚠️  Cache miss: %s", key)
	result, err := r.repo.GetAggregations(ctx, tenantID, startTime, endTime, granularity)
	if err != nil {
		return nil, err
	}

	_ = r.cache.Set(ctx, key, result)
	return result, nil
}

func (r *CachedAnalyticsRepository) SearchEvents(ctx context.Context, filters model.QueryFilters) ([]model.AnalyticsEvent, int64, error) {
	// Events search is typically not cached due to dynamic nature
	// But we can cache if needed with a shorter TTL
	key := cache.GenerateKey("events", filters.TenantID, filters.StartTime.Unix(), filters.EndTime.Unix(), filters.Limit, filters.Offset)

	type searchResult struct {
		Events []model.AnalyticsEvent
		Total  int64
	}

	var cached searchResult
	err := r.cache.Get(ctx, key, &cached)
	if err == nil {
		log.Printf("✅ Cache hit: %s", key)
		return cached.Events, cached.Total, nil
	}

	log.Printf("⚠️  Cache miss: %s", key)
	events, total, err := r.repo.SearchEvents(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	_ = r.cache.Set(ctx, key, searchResult{Events: events, Total: total})
	return events, total, nil
}

// InvalidateCache clears cache for a tenant
func (r *CachedAnalyticsRepository) InvalidateCache(ctx context.Context, tenantID string) error {
	pattern := fmt.Sprintf("analytics:*:%s:*", tenantID)
	return r.cache.DeletePattern(ctx, pattern)
}
