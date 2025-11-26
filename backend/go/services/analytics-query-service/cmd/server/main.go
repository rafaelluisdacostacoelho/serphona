// ==============================================================================
// Analytics Query Service
// ==============================================================================
// Exposes read APIs for dashboards. Queries ClickHouse (metrics) + Postgres (configs).
// Multi-tenant (filtering by tenant_id).

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("Starting Analytics Query Service...")

	router := setupRouter()

	srv := &http.Server{
		Addr:         getEnv("HTTP_ADDR", ":8082"),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("Server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "analytics-query-service"})
	})

	v1 := router.Group("/api/v1")
	{
		// Dashboard metrics
		v1.GET("/metrics/overview", getOverviewMetrics)
		v1.GET("/metrics/calls", getCallMetrics)
		v1.GET("/metrics/sentiment", getSentimentMetrics)
		v1.GET("/metrics/topics", getTopicMetrics)
		v1.GET("/metrics/agents", getAgentMetrics)

		// Time series
		v1.GET("/timeseries/calls", getCallTimeSeries)
		v1.GET("/timeseries/sentiment", getSentimentTimeSeries)

		// Aggregations
		v1.GET("/aggregations/hourly", getHourlyAggregations)
		v1.GET("/aggregations/daily", getDailyAggregations)

		// Search & Filter
		v1.POST("/search/events", searchEvents)
	}

	return router
}

// ==============================================================================
// Dashboard Metrics Handlers
// ==============================================================================

func getOverviewMetrics(c *gin.Context) {
	// TODO: Query ClickHouse for overview metrics
	c.JSON(http.StatusOK, gin.H{
		"total_calls":     0,
		"total_duration":  0,
		"avg_sentiment":   0.0,
		"resolution_rate": 0.0,
		"active_agents":   0,
		"period":          "last_30d",
	})
}

func getCallMetrics(c *gin.Context) {
	// TODO: Query ClickHouse for call metrics
	c.JSON(http.StatusOK, gin.H{
		"total":        0,
		"completed":    0,
		"abandoned":    0,
		"avg_duration": 0,
	})
}

func getSentimentMetrics(c *gin.Context) {
	// TODO: Query ClickHouse for sentiment distribution
	c.JSON(http.StatusOK, gin.H{
		"positive":  0,
		"neutral":   0,
		"negative":  0,
		"avg_score": 0.0,
	})
}

func getTopicMetrics(c *gin.Context) {
	// TODO: Query ClickHouse for topic distribution
	c.JSON(http.StatusOK, gin.H{
		"topics": []gin.H{},
	})
}

func getAgentMetrics(c *gin.Context) {
	// TODO: Query ClickHouse for agent performance
	c.JSON(http.StatusOK, gin.H{
		"agents": []gin.H{},
	})
}

// ==============================================================================
// Time Series Handlers
// ==============================================================================

func getCallTimeSeries(c *gin.Context) {
	// TODO: Query ClickHouse for time series data
	c.JSON(http.StatusOK, gin.H{
		"data":        []gin.H{},
		"granularity": "hourly",
	})
}

func getSentimentTimeSeries(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data":        []gin.H{},
		"granularity": "hourly",
	})
}

// ==============================================================================
// Aggregation Handlers
// ==============================================================================

func getHourlyAggregations(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"aggregations": []gin.H{},
	})
}

func getDailyAggregations(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"aggregations": []gin.H{},
	})
}

// ==============================================================================
// Search Handlers
// ==============================================================================

func searchEvents(c *gin.Context) {
	// TODO: Implement event search with filters
	c.JSON(http.StatusOK, gin.H{
		"events": []gin.H{},
		"total":  0,
		"page":   1,
		"limit":  50,
	})
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
