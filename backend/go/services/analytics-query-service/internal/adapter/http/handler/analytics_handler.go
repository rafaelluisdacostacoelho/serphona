package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/serphona/serphona/backend/go/services/analytics-query-service/internal/domain/model"
	"github.com/serphona/serphona/backend/go/services/analytics-query-service/internal/usecase"
)

type AnalyticsHandler struct {
	service *usecase.AnalyticsService
}

func NewAnalyticsHandler(service *usecase.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{service: service}
}

func (h *AnalyticsHandler) GetOverviewMetrics(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metrics, err := h.service.GetOverviewMetrics(c.Request.Context(), tenantID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetCallMetrics(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metrics, err := h.service.GetCallMetrics(c.Request.Context(), tenantID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetSentimentMetrics(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metrics, err := h.service.GetSentimentMetrics(c.Request.Context(), tenantID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetTopicMetrics(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	topics, err := h.service.GetTopicMetrics(c.Request.Context(), tenantID, startTime, endTime, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"topics": topics})
}

func (h *AnalyticsHandler) GetAgentMetrics(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agents, err := h.service.GetAgentMetrics(c.Request.Context(), tenantID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

func (h *AnalyticsHandler) GetCallTimeSeries(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	granularity := c.DefaultQuery("granularity", "hourly")
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data, err := h.service.GetCallTimeSeries(c.Request.Context(), tenantID, startTime, endTime, granularity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *AnalyticsHandler) GetSentimentTimeSeries(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	granularity := c.DefaultQuery("granularity", "hourly")
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data, err := h.service.GetSentimentTimeSeries(c.Request.Context(), tenantID, startTime, endTime, granularity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *AnalyticsHandler) GetAggregations(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	granularity := c.DefaultQuery("granularity", "hourly")
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.service.GetAggregations(c.Request.Context(), tenantID, startTime, endTime, granularity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"aggregations": results})
}

func (h *AnalyticsHandler) SearchEvents(c *gin.Context) {
	var filters model.QueryFilters
	if err := c.ShouldBindJSON(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	events, total, err := h.service.SearchEvents(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"page":   filters.Offset/filters.Limit + 1,
		"limit":  filters.Limit,
	})
}

func parseTimeRange(c *gin.Context) (time.Time, time.Time, error) {
	startStr := c.Query("start_time")
	endStr := c.Query("end_time")

	if startStr == "" || endStr == "" {
		// Default to last 7 days
		endTime := time.Now()
		startTime := endTime.AddDate(0, 0, -7)
		return startTime, endTime, nil
	}

	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	endTime, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return startTime, endTime, nil
}
