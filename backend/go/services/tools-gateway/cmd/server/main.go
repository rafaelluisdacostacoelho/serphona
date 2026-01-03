// ==============================================================================
// Tools Gateway Service
// ==============================================================================
// Manages registration and execution of Tools (external APIs).

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
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/adapter/http/handler"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/adapter/http/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/service"
	postgresrepo "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/infrastructure/repository/postgres"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/usecase"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.Println("Starting Tools Gateway Service...")

	// Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize dependencies
	toolRepo := postgresrepo.NewToolRepository(db)
	tenantToolRepo := postgresrepo.NewTenantToolRepository(db)
	executionRepo := postgresrepo.NewToolExecutionRepository(db)

	validator := service.NewSchemaValidator()
	serviceName := getEnv("SERVICE_NAME", "tools-gateway")
	serviceInstance := getEnv("SERVICE_INSTANCE", "tools-gateway-1")
	serviceAudience := getEnv("SERVICE_AUDIENCE", "")
	httpClient := service.NewHTTPClient(30*time.Second, serviceName, serviceInstance, serviceAudience)
	grpcClient := service.NewGRPCClient(serviceName, serviceInstance, serviceAudience)
	log.Printf("Service identity: %s/%s audience=%s", serviceName, serviceInstance, serviceAudience)

	toolService := usecase.NewToolService(toolRepo, validator)
	executorService := usecase.NewToolExecutorService(toolRepo, tenantToolRepo, executionRepo, validator, httpClient, grpcClient)

	// Initialize handlers
	toolHandler := handler.NewToolHandler(toolService, executorService)

	router := setupRouter(toolHandler)

	srv := &http.Server{
		Addr:         getEnv("HTTP_ADDR", ":8085"),
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

func initDB() (*gorm.DB, error) {
	dsn := getEnv("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/serphona_tools?sslmode=disable")
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func setupRouter(toolHandler *handler.ToolHandler) *gin.Engine {
	router := gin.Default()
	router.Use(middleware.Metrics())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tools-gateway"})
	})

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.HandlerFor(middleware.MetricsGatherer(), promhttp.HandlerOpts{})))

	// Mock auth middleware (replace with real auth later)
	authMiddleware := func(c *gin.Context) {
		// Mock tenant and user IDs for development
		c.Set("tenant_id", uuid.New())
		c.Set("user_id", uuid.New())
		c.Next()
	}

	v1 := router.Group("/api/v1")
	{
		// Tool management
		tools := v1.Group("/tools")
		{
			tools.GET("", toolHandler.ListTools)
			tools.POST("", toolHandler.CreateTool)
			tools.GET("/:id", toolHandler.GetTool)
			tools.POST("/:id/execute", authMiddleware, toolHandler.ExecuteTool)
		}
	}

	return router
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
