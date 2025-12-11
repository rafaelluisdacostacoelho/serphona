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
	"github.com/redis/go-redis/v9"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/adapter/http/handler"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/infrastructure/llm"
	redisRepo "github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/infrastructure/repository/redis"
	"github.com/serphona/serphona/backend/go/services/agent-orchestrator/internal/usecase"
)

func main() {
	log.Println("🚀 Starting Agent Orchestrator Service...")

	// Load environment variables
	config := loadConfig()

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}
	log.Println("✅ Connected to Redis")

	// Initialize LLM Client Pool
	clientPool := llm.SetupDefaultClients(config.OpenAIAPIKey)
	log.Println("✅ LLM Client Pool initialized")

	// Initialize repositories
	sessionRepo := redisRepo.NewSessionRepository(redisClient, 24*time.Hour)
	log.Println("✅ Session Repository initialized")

	// Initialize services
	sessionService := usecase.NewSessionService(sessionRepo)

	// Agent repository is optional (PostgreSQL required)
	// For now, using nil - agents will default to first available
	// To enable: Set DATABASE_URL and run migrations
	agentService := usecase.NewAgentService(nil)

	messageProcessing := usecase.NewMessageProcessingService(sessionService, agentService, clientPool)
	log.Println("✅ Services initialized")

	// Initialize handlers
	sessionHandler := handler.NewSessionHandler(sessionService, messageProcessing)
	log.Println("✅ Handlers initialized")

	// Setup HTTP router
	router := setupRouter(sessionHandler)

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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Printf("⚠️  Failed to close Redis connection: %v", err)
	}

	log.Println("👋 Server exited gracefully")
}

// Config holds application configuration
type Config struct {
	HTTPAddr        string
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	OpenAIAPIKey    string
	AnthropicAPIKey string
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	return Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:         0,
		OpenAIAPIKey:    getEnv("OPENAI_API_KEY", ""),
		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
	}
}

// setupRouter configures HTTP routes
func setupRouter(sessionHandler *handler.SessionHandler) *gin.Engine {
	// Set Gin mode
	if getEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware
	router.Use(corsMiddleware())
	router.Use(requestIDMiddleware())

	// Health check
	router.GET("/health", healthCheckHandler)
	router.GET("/ready", readinessHandler)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Session management
		sessions := v1.Group("/sessions")
		{
			sessions.POST("", sessionHandler.CreateSession)
			sessions.GET("/:id", sessionHandler.GetSession)
			sessions.DELETE("/:id", sessionHandler.EndSession)
			sessions.POST("/:id/messages", sessionHandler.SendMessage)
			sessions.GET("/:id/messages", sessionHandler.GetMessages)
		}

		// TODO: Agent management endpoints
		// agents := v1.Group("/agents")
		// {
		// 	agents.POST("", agentHandler.CreateAgent)
		// 	agents.GET("/:id", agentHandler.GetAgent)
		// 	agents.PUT("/:id", agentHandler.UpdateAgent)
		// 	agents.DELETE("/:id", agentHandler.DeleteAgent)
		// }
	}

	return router
}

// ==============================================================================
// Middleware
// ==============================================================================

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

// ==============================================================================
// Health Check Handlers
// ==============================================================================

func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "agent-orchestrator",
		"timestamp": time.Now().UTC(),
	})
}

func readinessHandler(c *gin.Context) {
	// TODO: Add actual readiness checks (Redis, DB, etc)
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"checks": gin.H{
			"redis": "ok",
			"llm":   "ok",
		},
	})
}

// ==============================================================================
// Helpers
// ==============================================================================

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
