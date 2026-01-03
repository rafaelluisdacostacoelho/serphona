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

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/adapter/http/handler"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/domain/repository"
	toolsHTTP "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/infrastructure/http"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/infrastructure/llm"
	postgresRepo "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/infrastructure/repository/postgres"
	redisRepo "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/infrastructure/repository/redis"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/agent-orchestrator/internal/usecase"
	"github.com/redis/go-redis/v9"
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

	// Initialize PostgreSQL (optional)
	var db *sql.DB
	var agentRepo repository.AgentRepository
	if config.DatabaseURL != "" {
		var err error
		db, err = sql.Open("postgres", config.DatabaseURL)
		if err != nil {
			log.Fatalf("❌ Failed to connect to PostgreSQL: %v", err)
		}

		// Test database connection
		if err := db.PingContext(ctx); err != nil {
			log.Fatalf("❌ Failed to ping PostgreSQL: %v", err)
		}
		log.Println("✅ Connected to PostgreSQL")

		// Create agent repository
		agentRepo = postgresRepo.NewAgentRepository(db)
		log.Println("✅ Agent Repository initialized (PostgreSQL)")
	} else {
		log.Println("⚠️  PostgreSQL not configured - agents will use default behavior")
		log.Println("   Set DATABASE_URL to enable agent persistence")
		agentRepo = nil
	}

	// Initialize Tools Gateway HTTP Client (optional)
	var toolsClient = toolsHTTP.NewToolsClient(config.ToolsGatewayURL, config.ToolsGatewayTok, config.ServiceAudience, config.ServiceName, config.ServiceInstance)
	if config.ToolsGatewayURL != "" {
		log.Printf("✅ Tools Gateway client initialized (%s)", config.ToolsGatewayURL)
	} else {
		log.Println("⚠️  Tools Gateway not configured - tool execution disabled")
		log.Println("   Set TOOLS_GATEWAY_URL to enable tool integration")
	}

	// Initialize LLM Client Pool
	clientPool := llm.SetupDefaultClients(config.OpenAIAPIKey)
	log.Println("✅ LLM Client Pool initialized")

	// Initialize repositories
	sessionRepo := redisRepo.NewSessionRepository(redisClient, 24*time.Hour)
	log.Println("✅ Session Repository initialized")

	// Initialize services
	sessionService := usecase.NewSessionService(sessionRepo)
	agentService := usecase.NewAgentService(agentRepo)
	messageProcessing := usecase.NewMessageProcessingService(sessionService, agentService, clientPool, toolsClient)
	log.Println("✅ Services initialized")

	// Initialize handlers
	sessionHandler := handler.NewSessionHandler(sessionService, messageProcessing)
	agentHandler := handler.NewAgentHandler(agentService)
	log.Println("✅ Handlers initialized")

	// Setup HTTP router
	router := setupRouter(sessionHandler, agentHandler)

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

	// Close connections
	if db != nil {
		if err := db.Close(); err != nil {
			log.Printf("⚠️  Failed to close PostgreSQL connection: %v", err)
		}
	}

	if err := redisClient.Close(); err != nil {
		log.Printf("⚠️  Failed to close Redis connection: %v", err)
	}

	log.Println("👋 Server exited gracefully")
}

// Config holds application configuration
type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	ToolsGatewayURL string
	ToolsGatewayTok string
	ServiceAudience string
	ServiceName     string
	ServiceInstance string
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
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		ToolsGatewayURL: getEnv("TOOLS_GATEWAY_URL", ""),
		ToolsGatewayTok: getEnv("TOOLS_GATEWAY_TOKEN", ""),
		ServiceAudience: getEnv("SERVICE_AUDIENCE", ""),
		ServiceName:     getEnv("SERVICE_NAME", "agent-orchestrator"),
		ServiceInstance: getEnv("SERVICE_INSTANCE", "agent-orchestrator-1"),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:         0,
		OpenAIAPIKey:    getEnv("OPENAI_API_KEY", ""),
		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
	}
}

// setupRouter configures HTTP routes
func setupRouter(sessionHandler *handler.SessionHandler, agentHandler *handler.AgentHandler) *gin.Engine {
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

		// Agent management
		agents := v1.Group("/agents")
		{
			agents.POST("", agentHandler.CreateAgent)
			agents.GET("", agentHandler.ListAgents)
			agents.GET("/:id", agentHandler.GetAgent)
			agents.PUT("/:id", agentHandler.UpdateAgent)
			agents.DELETE("/:id", agentHandler.DeleteAgent)
			agents.POST("/:id/activate", agentHandler.ActivateAgent)
			agents.POST("/:id/deactivate", agentHandler.DeactivateAgent)
		}
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
