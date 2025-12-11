package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
	"github.com/serphona/serphona/backend/go/services/analytics-query-service/internal/adapter/http/handler"
	"github.com/serphona/serphona/backend/go/services/analytics-query-service/internal/infrastructure/repository/clickhouse"
	"github.com/serphona/serphona/backend/go/services/analytics-query-service/internal/usecase"
)

func main() {
	log.Println("🚀 Starting Analytics Query Service...")

	// Load configuration
	config := loadConfig()

	// Initialize ClickHouse
	db, err := sql.Open("clickhouse", config.ClickHouseURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to ClickHouse: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping ClickHouse: %v", err)
	}
	log.Println("✅ Connected to ClickHouse")

	// Initialize repository and service
	analyticsRepo := clickhouse.NewAnalyticsRepository(db)
	analyticsService := usecase.NewAnalyticsService(analyticsRepo)
	log.Println("✅ Services initialized")

	// Initialize handler
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	log.Println("✅ Handlers initialized")

	// Setup HTTP router
	router := setupRouter(analyticsHandler)

	// Server configuration
	srv := &http.Server{
		Addr:         config.HTTPAddr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🌐 Server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("👋 Server exited gracefully")
}

type Config struct {
	HTTPAddr      string
	ClickHouseURL string
}

func loadConfig() Config {
	clickhouseHost := getEnv("CLICKHOUSE_HOST", "localhost")
	clickhousePort := getEnv("CLICKHOUSE_PORT", "9000")
	clickhouseDB := getEnv("CLICKHOUSE_DATABASE", "serphona_analytics")
	clickhouseUser := getEnv("CLICKHOUSE_USER", "default")
	clickhousePassword := getEnv("CLICKHOUSE_PASSWORD", "")

	clickhouseURL := "clickhouse://" + clickhouseHost + ":" + clickhousePort + "/" + clickhouseDB
	if clickhouseUser != "" {
		clickhouseURL += "?username=" + clickhouseUser
		if clickhousePassword != "" {
			clickhouseURL += "&password=" + clickhousePassword
		}
	}

	return Config{
		HTTPAddr:      getEnv("HTTP_ADDR", ":8084"),
		ClickHouseURL: clickhouseURL,
	}
}

func setupRouter(analyticsHandler *handler.AnalyticsHandler) *gin.Engine {
	if getEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", healthCheckHandler)
	router.GET("/ready", readinessHandler)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Dashboard metrics
		v1.GET("/metrics/overview", analyticsHandler.GetOverviewMetrics)
		v1.GET("/metrics/calls", analyticsHandler.GetCallMetrics)
		v1.GET("/metrics/sentiment", analyticsHandler.GetSentimentMetrics)
		v1.GET("/metrics/topics", analyticsHandler.GetTopicMetrics)
		v1.GET("/metrics/agents", analyticsHandler.GetAgentMetrics)

		// Time series
		v1.GET("/timeseries/calls", analyticsHandler.GetCallTimeSeries)
		v1.GET("/timeseries/sentiment", analyticsHandler.GetSentimentTimeSeries)

		// Aggregations
		v1.GET("/aggregations/hourly", func(c *gin.Context) {
			c.Request.URL.RawQuery += "&granularity=hourly"
			analyticsHandler.GetAggregations(c)
		})
		v1.GET("/aggregations/daily", func(c *gin.Context) {
			c.Request.URL.RawQuery += "&granularity=daily"
			analyticsHandler.GetAggregations(c)
		})

		// Search & Filter
		v1.POST("/search/events", analyticsHandler.SearchEvents)
	}

	return router
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "analytics-query-service",
		"timestamp": time.Now().UTC(),
	})
}

func readinessHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"checks": gin.H{
			"clickhouse": "ok",
		},
	})
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
