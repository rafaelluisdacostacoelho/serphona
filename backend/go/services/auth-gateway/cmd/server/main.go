package main

// @title Auth Gateway API
// @version 1.0
// @BasePath /api/v1

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	docs "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/docs"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/adapter/http/handler"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/adapter/http/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/adapter/oauth"
	postgresadapter "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/adapter/postgres"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/config"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/domain/user"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/service/jwt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/service/tenant"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/usecase/auth"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Swagger metadata for UI
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Title = "Auth Gateway API"
	docs.SwaggerInfo.Version = "1.0"

	logger.Info("Starting auth-gateway service",
		zap.String("env", cfg.Server.Env),
		zap.String("port", cfg.Server.Port),
	)

	// Initialize database
	db, err := initDatabase(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Auto-migrate tables
	if err := autoMigrate(db); err != nil {
		logger.Fatal("Failed to auto-migrate", zap.Error(err))
	}

	// Initialize services
	jwtService := jwt.NewService(
		cfg.JWT.SecretKey,
		cfg.JWT.AccessTokenDuration,
		cfg.JWT.RefreshTokenDuration,
	)

	userRepo := postgresadapter.NewUserRepository(db)
	tenantService := tenant.NewService(
		cfg.Outbound.TenantManagerURL,
		cfg.Service.Name,
		cfg.Service.Instance,
		cfg.Outbound.ServiceAuthToken,
		cfg.Service.Audience,
	)

	authUC := auth.NewUseCase(
		userRepo,
		jwtService,
		tenantService,
		cfg.JWT.AccessTokenDuration,
	)

	// Register OAuth providers
	if err := registerOAuthProviders(authUC, cfg.OAuth, logger); err != nil {
		logger.Error("Failed to register OAuth providers", zap.Error(err))
	}

	// Initialize HTTP handlers
	authHandler := handler.NewAuthHandler(authUC, jwtService, logger)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	// Setup router
	router := setupRouter(authHandler, authMiddleware, cfg, logger)

	// Start HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server",
			zap.String("address", srv.Addr),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited successfully")
}

// initDatabase initializes the database connection
func initDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// autoMigrate runs database migrations
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&user.User{},
		&user.Session{},
		&user.OAuthState{},
	)
}

// registerOAuthProviders registers OAuth providers
func registerOAuthProviders(authUC *auth.UseCase, cfg config.OAuthConfig, logger *zap.Logger) error {
	// Google OAuth
	if cfg.Google.Enabled && cfg.Google.ClientID != "" {
		googleProvider, err := oauth.NewGoogleProvider(
			cfg.Google.ClientID,
			cfg.Google.ClientSecret,
			cfg.Google.RedirectURL,
		)
		if err != nil {
			logger.Error("Failed to initialize Google OAuth", zap.Error(err))
		} else {
			authUC.RegisterOAuthProvider("google", googleProvider)
			logger.Info("Google OAuth provider registered")
		}
	}

	// Microsoft OAuth
	if cfg.Microsoft.Enabled && cfg.Microsoft.ClientID != "" {
		microsoftProvider, err := oauth.NewMicrosoftProvider(
			cfg.Microsoft.ClientID,
			cfg.Microsoft.ClientSecret,
			cfg.Microsoft.RedirectURL,
		)
		if err != nil {
			logger.Error("Failed to initialize Microsoft OAuth", zap.Error(err))
		} else {
			authUC.RegisterOAuthProvider("microsoft", microsoftProvider)
			logger.Info("Microsoft OAuth provider registered")
		}
	}

	return nil
}

// setupRouter sets up the Gin router with all routes
func setupRouter(authHandler *handler.AuthHandler, authMiddleware *middleware.AuthMiddleware, cfg *config.Config, logger *zap.Logger) *gin.Engine {
	// Set Gin mode
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Swagger UI (UI under /swagger/index.html, spec served from /swagger-docs/doc.json to avoid wildcard conflicts)
	router.GET("/swagger-docs/doc.json", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger-docs/doc.json")))

	// Middleware order: correlation -> logging -> metrics -> CORS -> recovery.
	router.Use(middleware.Correlation())
	router.Use(middleware.RequestLogger(logger))
	router.Use(middleware.Metrics())
	router.Use(middleware.CORS(cfg.Server.AllowedOrigins, cfg.Server.AllowCredentials))
	router.Use(gin.Recovery())

	// Health and metrics
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "auth-gateway",
		})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes
	api := router.Group("/api/v1")
	{
		// Public auth routes
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.RefreshToken)

			// OAuth routes
			authGroup.GET("/oauth/:provider", authHandler.GetOAuthURL)
			authGroup.GET("/oauth/:provider/callback", authHandler.HandleOAuthCallback)

			// Add CSRF protection middleware to public auth routes
			authGroup.Use(middleware.CSRFProtectionMiddleware())
		}

		// Protected auth routes
		protectedAuth := api.Group("/auth")
		protectedAuth.Use(authMiddleware.Authenticate())
		{
			protectedAuth.GET("/me", authHandler.GetCurrentUser)
			protectedAuth.POST("/logout", authHandler.Logout)
		}
	}

	return router
}
