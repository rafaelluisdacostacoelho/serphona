package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/domain/model"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/usecase"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// stubRepo satisfies AnalyticsRepository for contract tests.
type stubRepo struct{}

func (s *stubRepo) GetOverviewMetrics(_ context.Context, _ string, _ time.Time, _ time.Time) (*model.OverviewMetrics, error) {
	return nil, nil
}

func (s *stubRepo) GetCallMetrics(_ context.Context, _ string, _ time.Time, _ time.Time) (*model.CallMetrics, error) {
	return nil, nil
}

func (s *stubRepo) GetSentimentMetrics(_ context.Context, _ string, _ time.Time, _ time.Time) (*model.SentimentMetrics, error) {
	return nil, nil
}

func (s *stubRepo) GetTopicMetrics(_ context.Context, _ string, _ time.Time, _ time.Time, _ int) ([]model.TopicMetric, error) {
	return nil, nil
}

func (s *stubRepo) GetAgentMetrics(_ context.Context, _ string, _ time.Time, _ time.Time) ([]model.AgentMetric, error) {
	return nil, nil
}

func (s *stubRepo) GetCallTimeSeries(_ context.Context, _ string, _ time.Time, _ time.Time, _ string) (*model.TimeSeriesData, error) {
	return nil, nil
}

func (s *stubRepo) GetSentimentTimeSeries(_ context.Context, _ string, _ time.Time, _ time.Time, _ string) (*model.TimeSeriesData, error) {
	return nil, nil
}

func (s *stubRepo) GetAggregations(_ context.Context, _ string, _ time.Time, _ time.Time, _ string) ([]model.AggregationResult, error) {
	return nil, nil
}

func (s *stubRepo) SearchEvents(_ context.Context, _ model.QueryFilters) ([]model.AnalyticsEvent, int64, error) {
	return []model.AnalyticsEvent{{EventID: "e1"}, {EventID: "e2"}}, 2, nil
}

func TestSearchEventsEnvelopeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubRepo{}
	svc := usecase.NewAnalyticsService(repo)
	handler := NewAnalyticsHandler(svc)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	body := `{"start_time":"2024-01-01T00:00:00Z","end_time":"2024-01-08T00:00:00Z","limit":1,"offset":0}`
	req := httptest.NewRequest(http.MethodPost, "/analytics/events/search", bytes.NewBufferString(body))

	tracer := sdktrace.NewTracerProvider()
	traceCtx, span := tracer.Tracer("test").Start(req.Context(), "search")
	traceCtx = authmw.WithRequestID(traceCtx, "req-123")
	req = req.WithContext(traceCtx)
	span.End()

	ctx.Request = req
	ctx.Set("claims", &types.Claims{TenantID: "tenant-1"})

	handler.SearchEvents(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload struct {
		Data struct {
			Events []model.AnalyticsEvent `json:"events"`
			Total  int64                  `json:"total"`
			Page   int                    `json:"page"`
			Limit  int                    `json:"limit"`
		} `json:"data"`
		Meta response.Meta `json:"meta"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if payload.Meta.RequestID != "req-123" {
		t.Fatalf("expected request_id req-123, got %s", payload.Meta.RequestID)
	}
	if payload.Meta.TraceID == "" {
		t.Fatalf("expected trace_id to be populated")
	}
	if payload.Meta.Pagination == nil || payload.Meta.Pagination.PageSize != 1 || payload.Meta.Pagination.Total != 2 {
		t.Fatalf("unexpected pagination meta: %#v", payload.Meta.Pagination)
	}
	if len(payload.Data.Events) != 2 || payload.Data.Total != 2 || payload.Data.Page != 1 || payload.Data.Limit != 1 {
		t.Fatalf("unexpected data payload: %#v", payload.Data)
	}
}
