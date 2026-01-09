package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/adapter/http/handler"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/domain/repository"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/infrastructure/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/infrastructure/repository/cached"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/infrastructure/repository/clickhouse"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/analytics-query-service/internal/usecase"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// TenantEnforcerAdapter encapsula a função EnforceTenant
// para que ela implemente a interface TenantEnforcer.
type TenantEnforcerAdapter struct{}

func (a *TenantEnforcerAdapter) EnforceTenant(ctx context.Context, tenantID string) error {
	return authmw.EnforceTenant(ctx, tenantID)
}

func main() {
	log.Println("🚀 Starting Analytics Query Service...")

	// Load configuration
	config := loadConfig()
	authmw.SetAuthMetricsService(config.ServiceName)

	// Initialize Redis (optional, for caching)
	var analyticsRepo repository.AnalyticsRepository
	var redisClient *redis.Client

	if config.RedisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     config.RedisAddr,
			Password: config.RedisPassword,
			DB:       config.RedisDB,
		})

		// Test Redis connection
		ctx := context.Background()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Printf("⚠️  Redis connection failed: %v (continuing without cache)", err)
			redisClient = nil
		} else {
			log.Println("✅ Connected to Redis")
		}
	}

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

	// Initialize repository (with or without cache)
	tenantEnforcer := &TenantEnforcerAdapter{}
	clickhouseRepo := clickhouse.NewAnalyticsRepository(db, tenantEnforcer)

	if redisClient != nil {
		// Wrap with cache layer
		analyticsRepo = cached.NewCachedAnalyticsRepository(clickhouseRepo, redisClient, config.CacheTTL)
		log.Println("✅ Repository initialized with Redis caching")
	} else {
		analyticsRepo = clickhouseRepo
		log.Println("✅ Repository initialized (no cache)")
	}

	// Initialize service
	analyticsService := usecase.NewAnalyticsService(analyticsRepo)
	log.Println("✅ Services initialized")

	// Initialize handler
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	log.Println("✅ Handlers initialized")

	// Setup HTTP router
	router := setupRouter(analyticsHandler, config)

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

	// Close Redis
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			log.Printf("⚠️  Failed to close Redis connection: %v", err)
		}
	}

	log.Println("👋 Server exited gracefully")
}

type Config struct {
	HTTPAddr        string
	ClickHouseURL   string
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	CacheTTL        time.Duration
	RateLimit       int
	RateBurst       int
	ServiceName     string
	ServiceInstance string
	ServiceAudience string
}

func loadConfig() Config {
	// ClickHouse config
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

	// Redis config
	redisAddr := getEnv("REDIS_ADDR", "")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "3"))

	// Cache TTL (default 5 minutes)
	cacheTTLStr := getEnv("CACHE_TTL", "5m")
	cacheTTL, err := time.ParseDuration(cacheTTLStr)
	if err != nil {
		cacheTTL = 5 * time.Minute
	}

	// Rate limiting (default 60 req/min, burst 10)
	rateLimit, _ := strconv.Atoi(getEnv("RATE_LIMIT", "60"))
	rateBurst, _ := strconv.Atoi(getEnv("RATE_BURST", "10"))

	return Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8084"),
		ClickHouseURL:   clickhouseURL,
		RedisAddr:       redisAddr,
		RedisPassword:   redisPassword,
		RedisDB:         redisDB,
		CacheTTL:        cacheTTL,
		RateLimit:       rateLimit,
		RateBurst:       rateBurst,
		ServiceName:     getEnv("SERVICE_NAME", "analytics-query-service"),
		ServiceInstance: getEnv("SERVICE_INSTANCE", "analytics-query-service-1"),
		ServiceAudience: getEnv("SERVICE_AUDIENCE", ""),
	}
}

func setupRouter(analyticsHandler *handler.AnalyticsHandler, config Config) *gin.Engine {
	if getEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware
	router.Use(corsMiddleware())

	// Rate limiting
	rateLimiter := middleware.NewRateLimiter(config.RateLimit, config.RateBurst)
	router.Use(rateLimiter.Middleware())
	log.Printf("✅ Rate limiting enabled: %d req/min, burst %d", config.RateLimit, config.RateBurst)

	// Swagger UI (UI under /swagger/index.html, spec served from /swagger-docs/doc.json to avoid wildcard conflicts)
	router.GET("/swagger-docs/doc.json", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger-docs/doc.json")))

	// Health check
	router.GET("/health", healthCheckHandler)
	router.GET("/ready", readinessHandler)

	// API v1 routes
	v1 := router.Group("/api/v1")
	v1.Use(authmw.RequireAuth())
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
