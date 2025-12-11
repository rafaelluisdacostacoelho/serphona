package model

import "time"

// OverviewMetrics represents high-level dashboard metrics
type OverviewMetrics struct {
	TotalCalls       int64             `json:"total_calls"`
	TotalDuration    int64             `json:"total_duration"` // seconds
	AvgSentiment     float64           `json:"avg_sentiment"`
	ResolutionRate   float64           `json:"resolution_rate"`
	ActiveAgents     int               `json:"active_agents"`
	Period           string            `json:"period"`
	ComparisonPeriod *PeriodComparison `json:"comparison,omitempty"`
}

// PeriodComparison compares metrics with previous period
type PeriodComparison struct {
	CallsChange      float64 `json:"calls_change_percent"`
	DurationChange   float64 `json:"duration_change_percent"`
	SentimentChange  float64 `json:"sentiment_change_percent"`
	ResolutionChange float64 `json:"resolution_change_percent"`
}

// CallMetrics represents call-specific metrics
type CallMetrics struct {
	Total       int64   `json:"total"`
	Completed   int64   `json:"completed"`
	Abandoned   int64   `json:"abandoned"`
	AvgDuration float64 `json:"avg_duration"`  // seconds
	AvgWaitTime float64 `json:"avg_wait_time"` // seconds
	PeakHour    string  `json:"peak_hour,omitempty"`
}

// SentimentMetrics represents sentiment distribution
type SentimentMetrics struct {
	Positive int64   `json:"positive"`
	Neutral  int64   `json:"neutral"`
	Negative int64   `json:"negative"`
	AvgScore float64 `json:"avg_score"`
	Total    int64   `json:"total"`
}

// TopicMetric represents a single topic with its metrics
type TopicMetric struct {
	Topic      string  `json:"topic"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
	AvgScore   float64 `json:"avg_score,omitempty"`
}

// AgentMetric represents agent performance metrics
type AgentMetric struct {
	AgentID        string  `json:"agent_id"`
	AgentName      string  `json:"agent_name"`
	TotalCalls     int64   `json:"total_calls"`
	AvgDuration    float64 `json:"avg_duration"`
	AvgSentiment   float64 `json:"avg_sentiment"`
	ResolutionRate float64 `json:"resolution_rate"`
	Utilization    float64 `json:"utilization"` // percentage
}

// TimeSeriesPoint represents a point in time series
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitempty"`
}

// TimeSeriesData represents time series with metadata
type TimeSeriesData struct {
	Data        []TimeSeriesPoint `json:"data"`
	Granularity string            `json:"granularity"` // hourly, daily, weekly
	Metric      string            `json:"metric"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
}

// AggregationResult represents aggregated data
type AggregationResult struct {
	Timestamp    time.Time              `json:"timestamp"`
	TotalCalls   int64                  `json:"total_calls"`
	AvgDuration  float64                `json:"avg_duration"`
	AvgSentiment float64                `json:"avg_sentiment"`
	TopTopics    []string               `json:"top_topics,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AnalyticsEvent represents a single analytics event
type AnalyticsEvent struct {
	EventID   string                 `json:"event_id"`
	TenantID  string                 `json:"tenant_id"`
	SessionID string                 `json:"session_id"`
	UserID    string                 `json:"user_id,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  int64                  `json:"duration,omitempty"`
	Sentiment float64                `json:"sentiment,omitempty"`
	Topics    []string               `json:"topics,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// QueryFilters represents search/filter parameters
type QueryFilters struct {
	TenantID  string    `json:"tenant_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	AgentID   string    `json:"agent_id,omitempty"`
	EventType string    `json:"event_type,omitempty"`
	MinScore  *float64  `json:"min_score,omitempty"`
	MaxScore  *float64  `json:"max_score,omitempty"`
	Topics    []string  `json:"topics,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}
