package main

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
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/adapter/http/handler"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/adapter/http/middleware"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/adapter/oauth"
	postgresadapter "github.com/serphona/serphona/backend/go/services/auth-gateway/internal/adapter/postgres"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/config"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/domain/user"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/service/jwt"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/service/tenant"
	"github.com/serphona/serphona/backend/go/services/auth-gateway/internal/usecase/auth"
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
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

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
	tenantService := tenant.NewService("http://localhost:8081") // TODO: Get from config

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
	router := setupRouter(authHandler, authMiddleware, cfg)

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

	// Apple OAuth
	if cfg.Apple.Enabled && cfg.Apple.ClientID != "" {
		appleProvider, err := oauth.NewAppleProvider(
			cfg.Apple.ClientID,
			"", // teamID
			"", // keyID
			"", // privateKey
			cfg.Apple.RedirectURL,
		)
		if err != nil {
			logger.Error("Failed to initialize Apple OAuth", zap.Error(err))
		} else {
			authUC.RegisterOAuthProvider("apple", appleProvider)
			logger.Info("Apple OAuth provider registered")
		}
	}

	return nil
}

// setupRouter sets up the Gin router with all routes
func setupRouter(authHandler *handler.AuthHandler, authMiddleware *middleware.AuthMiddleware, cfg *config.Config) *gin.Engine {
	// Set Gin mode
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware
	router.Use(middleware.CORS())
	router.Use(gin.Recovery())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "auth-gateway",
		})
	})

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
