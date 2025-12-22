package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/domain/model"
)

type AnalyticsRepository struct {
	db *sql.DB
}

func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) GetOverviewMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.OverviewMetrics, error) {
	query := `
		SELECT 
			count() as total_calls,
			sum(duration) as total_duration,
			avg(sentiment_score) as avg_sentiment,
			countIf(resolution_status = 'resolved') / count() as resolution_rate,
			uniqExact(agent_id) as active_agents
		FROM analytics_events
		WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?
	`

	var metrics model.OverviewMetrics
	err := r.db.QueryRowContext(ctx, query, tenantID, startTime, endTime).Scan(
		&metrics.TotalCalls,
		&metrics.TotalDuration,
		&metrics.AvgSentiment,
		&metrics.ResolutionRate,
		&metrics.ActiveAgents,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get overview metrics: %w", err)
	}

	metrics.Period = fmt.Sprintf("%s to %s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
	return &metrics, nil
}

func (r *AnalyticsRepository) GetCallMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.CallMetrics, error) {
	query := `
		SELECT 
			count() as total,
			countIf(status = 'completed') as completed,
			countIf(status = 'abandoned') as abandoned,
			avg(duration) as avg_duration,
			avg(wait_time) as avg_wait_time
		FROM analytics_events
		WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?
		AND event_type = 'call'
	`

	var metrics model.CallMetrics
	err := r.db.QueryRowContext(ctx, query, tenantID, startTime, endTime).Scan(
		&metrics.Total,
		&metrics.Completed,
		&metrics.Abandoned,
		&metrics.AvgDuration,
		&metrics.AvgWaitTime,
	)

	return &metrics, err
}

func (r *AnalyticsRepository) GetSentimentMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.SentimentMetrics, error) {
	query := `
		SELECT 
			countIf(sentiment_label = 'positive') as positive,
			countIf(sentiment_label = 'neutral') as neutral,
			countIf(sentiment_label = 'negative') as negative,
			avg(sentiment_score) as avg_score,
			count() as total
		FROM analytics_events
		WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?
	`

	var metrics model.SentimentMetrics
	err := r.db.QueryRowContext(ctx, query, tenantID, startTime, endTime).Scan(
		&metrics.Positive,
		&metrics.Neutral,
		&metrics.Negative,
		&metrics.AvgScore,
		&metrics.Total,
	)

	return &metrics, err
}

func (r *AnalyticsRepository) GetTopicMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time, limit int) ([]model.TopicMetric, error) {
	query := `
		SELECT 
			topic,
			count() as count,
			(count() * 100.0) / (SELECT count() FROM analytics_events WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?) as percentage,
			avg(sentiment_score) as avg_score
		FROM analytics_events
		ARRAY JOIN topics as topic
		WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY topic
		ORDER BY count DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, startTime, endTime, tenantID, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []model.TopicMetric
	for rows.Next() {
		var t model.TopicMetric
		if err := rows.Scan(&t.Topic, &t.Count, &t.Percentage, &t.AvgScore); err != nil {
			return nil, err
		}
		topics = append(topics, t)
	}

	return topics, nil
}

func (r *AnalyticsRepository) GetAgentMetrics(ctx context.Context, tenantID string, startTime, endTime time.Time) ([]model.AgentMetric, error) {
	query := `
		SELECT 
			agent_id,
			any(agent_name) as agent_name,
			count() as total_calls,
			avg(duration) as avg_duration,
			avg(sentiment_score) as avg_sentiment,
			countIf(resolution_status = 'resolved') / count() as resolution_rate,
			(sum(duration) * 100.0) / ? as utilization
		FROM analytics_events
		WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?
		AND agent_id != ''
		GROUP BY agent_id
		ORDER BY total_calls DESC
	`

	totalSeconds := endTime.Sub(startTime).Seconds()
	rows, err := r.db.QueryContext(ctx, query, totalSeconds, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []model.AgentMetric
	for rows.Next() {
		var a model.AgentMetric
		if err := rows.Scan(&a.AgentID, &a.AgentName, &a.TotalCalls, &a.AvgDuration, &a.AvgSentiment, &a.ResolutionRate, &a.Utilization); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}

	return agents, nil
}

func (r *AnalyticsRepository) GetCallTimeSeries(ctx context.Context, tenantID string, startTime, endTime time.Time, granularity string) (*model.TimeSeriesData, error) {
	interval := getIntervalFromGranularity(granularity)
	query := fmt.Sprintf(`
		SELECT 
			toStartOfInterval(timestamp, INTERVAL %s) as bucket,
			count() as value
		FROM analytics_events
		WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?
		AND event_type = 'call'
		GROUP BY bucket
		ORDER BY bucket
	`, interval)

	rows, err := r.db.QueryContext(ctx, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []model.TimeSeriesPoint
	for rows.Next() {
		var p model.TimeSeriesPoint
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}

	return &model.TimeSeriesData{
		Data:        points,
		Granularity: granularity,
		Metric:      "calls",
		StartTime:   startTime,
		EndTime:     endTime,
	}, nil
}

func (r *AnalyticsRepository) GetSentimentTimeSeries(ctx context.Context, tenantID string, startTime, endTime time.Time, granularity string) (*model.TimeSeriesData, error) {
	interval := getIntervalFromGranularity(granularity)
	query := fmt.Sprintf(`
		SELECT 
			toStartOfInterval(timestamp, INTERVAL %s) as bucket,
			avg(sentiment_score) as value
		FROM analytics_events
		WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY bucket
		ORDER BY bucket
	`, interval)

	rows, err := r.db.QueryContext(ctx, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []model.TimeSeriesPoint
	for rows.Next() {
		var p model.TimeSeriesPoint
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}

	return &model.TimeSeriesData{
		Data:        points,
		Granularity: granularity,
		Metric:      "sentiment",
		StartTime:   startTime,
		EndTime:     endTime,
	}, nil
}

func (r *AnalyticsRepository) GetAggregations(ctx context.Context, tenantID string, startTime, endTime time.Time, granularity string) ([]model.AggregationResult, error) {
	interval := getIntervalFromGranularity(granularity)
	query := fmt.Sprintf(`
		SELECT 
			toStartOfInterval(timestamp, INTERVAL %s) as bucket,
			count() as total_calls,
			avg(duration) as avg_duration,
			avg(sentiment_score) as avg_sentiment
		FROM analytics_events
		WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY bucket
		ORDER BY bucket
	`, interval)

	rows, err := r.db.QueryContext(ctx, query, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.AggregationResult
	for rows.Next() {
		var r model.AggregationResult
		if err := rows.Scan(&r.Timestamp, &r.TotalCalls, &r.AvgDuration, &r.AvgSentiment); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (r *AnalyticsRepository) SearchEvents(ctx context.Context, filters model.QueryFilters) ([]model.AnalyticsEvent, int64, error) {
	// Build query dynamically based on filters
	query := `SELECT event_id, tenant_id, session_id, user_id, agent_id, event_type, timestamp, duration, sentiment_score, topics FROM analytics_events WHERE tenant_id = ?`
	args := []interface{}{filters.TenantID}

	query += ` AND timestamp >= ? AND timestamp <= ?`
	args = append(args, filters.StartTime, filters.EndTime)

	if filters.AgentID != "" {
		query += ` AND agent_id = ?`
		args = append(args, filters.AgentID)
	}

	if filters.EventType != "" {
		query += ` AND event_type = ?`
		args = append(args, filters.EventType)
	}

	if filters.Limit == 0 {
		filters.Limit = 50
	}

	query += ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []model.AnalyticsEvent
	for rows.Next() {
		var e model.AnalyticsEvent
		var topics string
		if err := rows.Scan(&e.EventID, &e.TenantID, &e.SessionID, &e.UserID, &e.AgentID, &e.EventType, &e.Timestamp, &e.Duration, &e.Sentiment, &topics); err != nil {
			return nil, 0, err
		}
		// Parse topics array
		events = append(events, e)
	}

	// Get total count
	var total int64
	countQuery := `SELECT count() FROM analytics_events WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?`
	_ = r.db.QueryRowContext(ctx, countQuery, filters.TenantID, filters.StartTime, filters.EndTime).Scan(&total)

	return events, total, nil
}

func getIntervalFromGranularity(granularity string) string {
	switch granularity {
	case "hourly":
		return "1 HOUR"
	case "daily":
		return "1 DAY"
	case "weekly":
		return "1 WEEK"
	default:
		return "1 HOUR"
	}
}
