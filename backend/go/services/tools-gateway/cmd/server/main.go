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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/adapter/http/handler"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/adapter/http/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/config"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/service"
	postgresrepo "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/infrastructure/repository/postgres"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/usecase"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.Println("Starting Tools Gateway Service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	configureAuth(cfg)

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
	serviceAudience := cfg.Auth.ServiceAudience
	authmw.SetAuthMetricsService(serviceName)
	httpClient := service.NewHTTPClient(30*time.Second, serviceName, serviceInstance, serviceAudience)
	grpcClient := service.NewGRPCClient(serviceName, serviceInstance, serviceAudience)
	log.Printf("Service identity: %s/%s audience=%s", serviceName, serviceInstance, serviceAudience)

	execPolicy := usecase.ExecutionPolicy{
		AllowedHosts:    cfg.Execution.AllowedHosts,
		MaxPayloadBytes: cfg.Execution.MaxPayloadBytes,
		AllowedMethods:  cfg.Execution.AllowedMethods,
		BlockedMethods:  cfg.Execution.BlockedMethods,
		AllowedHeaders:  cfg.Execution.AllowedHeaders,
		BlockedHeaders:  cfg.Execution.BlockedHeaders,
		MaxQueryParams:  cfg.Execution.MaxQueryParams,
	}

	toolService := usecase.NewToolService(toolRepo, validator, execPolicy)
	executorService := usecase.NewToolExecutorService(
		toolRepo,
		tenantToolRepo,
		executionRepo,
		validator,
		httpClient,
		grpcClient,
		execPolicy,
	)

	// Initialize handlers
	toolHandler := handler.NewToolHandler(toolService, executorService)

	router := setupRouter(toolHandler, serviceName)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
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

func configureAuth(cfg *config.Config) {
	authjwt.SetValidationConfig(authjwt.ValidationConfig{
		AllowedAlgs:    cfg.Auth.AllowedAlgs,
		Issuer:         cfg.Auth.Issuer,
		Audience:       cfg.Auth.Audience,
		ClockSkew:      cfg.Auth.ClockSkew,
		MaxTokenBytes:  cfg.Auth.MaxTokenBytes,
		JWKSURL:        cfg.Auth.JWKSURL,
		JWKSCacheTTL:   cfg.Auth.JWKSCacheTTL,
		AllowedKIDs:    cfg.Auth.AllowedKIDs,
		RequiredScopes: cfg.Auth.RequiredScopes,
	})
	if cfg.Auth.JWTSecret != "" {
		authjwt.SetSecret(cfg.Auth.JWTSecret)
	}
}

func setupRouter(toolHandler *handler.ToolHandler, serviceName string) *gin.Engine {
	router := gin.Default()
	authmw.SetMetricsRegisterer(middleware.MetricsRegisterer())
	router.Use(middleware.RequestLogger(nil))
	router.Use(middleware.Metrics(serviceName))
	router.Use(middleware.AuthMetrics(serviceName))

	// Swagger UI (UI under /swagger/index.html, spec served from /swagger-docs/doc.json to avoid wildcard conflicts)
	router.GET("/swagger-docs/doc.json", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger-docs/doc.json")))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tools-gateway"})
	})

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.HandlerFor(middleware.MetricsGatherer(), promhttp.HandlerOpts{})))

	v1 := router.Group("/api/v1")
	{
		// Tool management (authenticated)
		tools := v1.Group("/tools")
		tools.Use(authmw.RequireAuth())
		{
			tools.GET("", authmw.RequireScopes("tools:read"), toolHandler.ListTools)
			tools.POST("", authmw.RequireScopes("tools:write"), toolHandler.CreateTool)
			tools.GET("/:id", authmw.RequireScopes("tools:read"), toolHandler.GetTool)
			tools.POST("/:id/execute", authmw.RequireScopes("tools:execute"), toolHandler.ExecuteTool)
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
