package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/domain/model"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/usecase"
)

type AnalyticsHandler struct {
	service *usecase.AnalyticsService
}

func NewAnalyticsHandler(service *usecase.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{service: service}
}

func (h *AnalyticsHandler) GetOverviewMetrics(c *gin.Context) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		return
	}
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_TIME_RANGE", err.Error(), nil)
		return
	}

	metrics, err := h.service.GetOverviewMetrics(c.Request.Context(), tenantID, startTime, endTime)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetCallMetrics(c *gin.Context) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		return
	}
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_TIME_RANGE", err.Error(), nil)
		return
	}

	metrics, err := h.service.GetCallMetrics(c.Request.Context(), tenantID, startTime, endTime)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetSentimentMetrics(c *gin.Context) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		return
	}
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_TIME_RANGE", err.Error(), nil)
		return
	}

	metrics, err := h.service.GetSentimentMetrics(c.Request.Context(), tenantID, startTime, endTime)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetTopicMetrics(c *gin.Context) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		return
	}
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_TIME_RANGE", err.Error(), nil)
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	topics, err := h.service.GetTopicMetrics(c.Request.Context(), tenantID, startTime, endTime, limit)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{"topics": topics})
}

func (h *AnalyticsHandler) GetAgentMetrics(c *gin.Context) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		return
	}
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_TIME_RANGE", err.Error(), nil)
		return
	}

	agents, err := h.service.GetAgentMetrics(c.Request.Context(), tenantID, startTime, endTime)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{"agents": agents})
}

func (h *AnalyticsHandler) GetCallTimeSeries(c *gin.Context) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		return
	}
	granularity := c.DefaultQuery("granularity", "hourly")
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_TIME_RANGE", err.Error(), nil)
		return
	}

	data, err := h.service.GetCallTimeSeries(c.Request.Context(), tenantID, startTime, endTime, granularity)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, data)
}

func (h *AnalyticsHandler) GetSentimentTimeSeries(c *gin.Context) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		return
	}
	granularity := c.DefaultQuery("granularity", "hourly")
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_TIME_RANGE", err.Error(), nil)
		return
	}

	data, err := h.service.GetSentimentTimeSeries(c.Request.Context(), tenantID, startTime, endTime, granularity)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, data)
}

func (h *AnalyticsHandler) GetAggregations(c *gin.Context) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		return
	}
	granularity := c.DefaultQuery("granularity", "hourly")
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_TIME_RANGE", err.Error(), nil)
		return
	}

	results, err := h.service.GetAggregations(c.Request.Context(), tenantID, startTime, endTime, granularity)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{"aggregations": results})
}

func (h *AnalyticsHandler) SearchEvents(c *gin.Context) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		return
	}
	var filters model.QueryFilters
	if err := c.ShouldBindJSON(&filters); err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), nil)
		return
	}
	filters.TenantID = tenantID
	if filters.Limit <= 0 {
		filters.Limit = 50
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	events, total, err := h.service.SearchEvents(c.Request.Context(), filters)
	if err != nil {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	page := filters.Offset/filters.Limit + 1
	totalPages := int(total) / filters.Limit
	if int(total)%filters.Limit != 0 {
		totalPages++
	}

	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"page":   page,
		"limit":  filters.Limit,
	}, response.WithPagination(response.Pagination{Page: page, PageSize: filters.Limit, Total: int(total), TotalPages: totalPages}))
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

func tenantIDFromContext(c *gin.Context) (string, bool) {
	tenantID, err := authmw.GetTenantIDFromContext(c)
	if err != nil || tenantID == "" {
		response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context required", nil)
		return "", false
	}
	c.Request.Header = authmw.EnsureTenantHeader(c.Request.Header, tenantID)
	ctxWithTenant := authmw.WithTenantID(c.Request.Context(), tenantID)
	c.Request = c.Request.WithContext(ctxWithTenant)
	return tenantID, true
}
