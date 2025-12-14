package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tenant-manager/internal/adapter/grpc/handler"
	httpHandler "tenant-manager/internal/adapter/http/handler"
	"tenant-manager/internal/adapter/http/middleware"
	"tenant-manager/internal/adapter/kafka"
	"tenant-manager/internal/adapter/postgres"
	"tenant-manager/internal/adapter/redis"
	"tenant-manager/internal/application/tenant"
	"tenant-manager/internal/config"
	tenantDomain "tenant-manager/internal/domain/tenant"
	"tenant-manager/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	serviceName    = "tenant-manager"
	serviceVersion = "1.0.0"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.New(cfg.LogLevel, cfg.Environment)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("Starting Tenant Manager Service",
		zap.String("service", serviceName),
		zap.String("version", serviceVersion),
		zap.String("environment", cfg.Environment),
	)

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize dependencies
	deps, cleanup, err := initializeDependencies(context.Background(), cfg, log)
	if err != nil {
		log.Fatal("Failed to initialize dependencies", zap.Error(err))
	}
	defer cleanup()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start gRPC server in goroutine
	grpcServer := startGRPCServer(cfg, deps, log)
	go func() {
		if err := serveGRPC(grpcServer, cfg.Server.GRPCPort, log); err != nil {
			log.Error("gRPC server error", zap.Error(err))
			cancel()
		}
	}()

	// Start HTTP server
	httpServer := startHTTPServer(cfg, deps, log)
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		log.Info("HTTP server listening", zap.String("address", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", zap.Error(err))
			cancel()
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Info("Shutdown signal received")
	case <-ctx.Done():
		log.Info("Context cancelled")
	}

	// Graceful shutdown
	gracefulShutdown(ctx, httpServer, grpcServer, log)
	log.Info("Service stopped gracefully")
}

// Dependencies holds all service dependencies
type Dependencies struct {
	// Repositories
	TenantRepo tenantDomain.Repository

	// Services
	TenantService *tenant.Service

	// Infrastructure
	DB             *postgres.DB
	Cache          *redis.Cache
	EventPublisher *kafka.EventPublisher
	Producer       *kafka.Producer
}

// initializeDependencies initializes all service dependencies
func initializeDependencies(ctx context.Context, cfg *config.Config, log *zap.Logger) (*Dependencies, func(), error) {
	deps := &Dependencies{}
	cleanupFuncs := []func(){}

	// Initialize PostgreSQL
	log.Info("Connecting to PostgreSQL", zap.String("url", maskPassword(cfg.Database.URL)))
	db, err := postgres.NewConnection(ctx, cfg.Database)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	deps.DB = &postgres.DB{Pool: db}
	cleanupFuncs = append(cleanupFuncs, func() {
		log.Info("Closing PostgreSQL connection")
		db.Close()
	})

	// Run migrations if enabled
	if cfg.Database.AutoMigrate {
		log.Info("Running database migrations")
		if err := postgres.RunMigrations(cfg.Database.URL, cfg.Database.MigrationsPath); err != nil {
			log.Warn("Failed to run migrations", zap.Error(err))
		}
	}

	// Initialize Redis
	log.Info("Connecting to Redis", zap.String("url", maskPassword(cfg.Redis.URL)))
	redisCache, err := redis.NewClient(ctx, cfg.Redis)
	if err != nil {
		log.Warn("Failed to connect to Redis, caching disabled", zap.Error(err))
		redisCache = nil
	} else {
		deps.Cache = redisCache
		cleanupFuncs = append(cleanupFuncs, func() {
			log.Info("Closing Redis connection")
			if err := redisCache.Close(); err != nil {
				log.Error("Error closing Redis", zap.Error(err))
			}
		})
	}

	// Initialize Kafka Producer
	log.Info("Connecting to Kafka", zap.Strings("brokers", cfg.Kafka.Brokers))
	producer, err := kafka.NewProducer(cfg.Kafka)
	if err != nil {
		log.Warn("Failed to connect to Kafka, events disabled", zap.Error(err))
		producer = nil
	} else {
		deps.Producer = producer
		deps.EventPublisher = kafka.NewEventPublisher(producer, cfg.Kafka.TopicPrefix)
		cleanupFuncs = append(cleanupFuncs, func() {
			log.Info("Closing Kafka connection")
			if err := producer.Close(); err != nil {
				log.Error("Error closing Kafka", zap.Error(err))
			}
		})
	}

	// Initialize repositories
	deps.TenantRepo = postgres.NewTenantRepository(db)

	// Initialize domain services
	tenantDomainService := tenantDomain.NewService(deps.TenantRepo)

	// Initialize application services
	var tenantCache tenantDomain.Cache
	if redisCache != nil {
		tenantCache = redis.NewTenantCache(redisCache, cfg.Redis.CacheTTL)
	}

	deps.TenantService = tenant.NewService(
		deps.TenantRepo,
		nil, // apiKeyRepo - not yet implemented
		tenantCache,
		deps.EventPublisher,
		log,
	)
	_ = tenantDomainService // Use domain service when needed

	cleanup := func() {
		for i := len(cleanupFuncs) - 1; i >= 0; i-- {
			cleanupFuncs[i]()
		}
	}

	log.Info("Dependencies initialized successfully")
	return deps, cleanup, nil
}

// startHTTPServer creates and configures the HTTP server
func startHTTPServer(cfg *config.Config, deps *Dependencies, log *zap.Logger) *http.Server {
	// Create Gin router
	r := gin.New()

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.ZapLogger(log))
	r.Use(middleware.CORS())

	// Health checks (no auth required)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": serviceName,
			"version": serviceVersion,
		})
	})

	r.GET("/ready", func(c *gin.Context) {
		if deps.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": "database not connected"})
			return
		}
		// Check DB connection
		if err := deps.DB.Pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// API routes with authentication
	api := r.Group("/api/v1")
	api.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		// Tenant routes
		tenantHandler := httpHandler.NewGinTenantHandler(deps.TenantService, log)
		api.POST("/tenants", tenantHandler.Create)
		api.GET("/tenants", tenantHandler.List)
		api.GET("/tenants/:id", tenantHandler.Get)
		api.PUT("/tenants/:id", tenantHandler.Update)
		api.DELETE("/tenants/:id", tenantHandler.Delete)

		// API Key routes - placeholder
		// apiKeyHandler := httpHandler.NewAPIKeyHandler(deps.APIKeyService)
		// api.POST("/tenants/:id/api-keys", apiKeyHandler.Create)
		// api.GET("/tenants/:id/api-keys", apiKeyHandler.List)
		// api.DELETE("/tenants/:id/api-keys/:keyId", apiKeyHandler.Delete)
	}

	// Create HTTP server
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:        r,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	return server
}

// startGRPCServer creates and configures the gRPC server
func startGRPCServer(cfg *config.Config, deps *Dependencies, log *zap.Logger) *grpc.Server {
	// Create gRPC server with interceptors
	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpcUnaryInterceptor(log)),
		grpc.StreamInterceptor(grpcStreamInterceptor(log)),
	)

	// Register services
	tenantHandler := handler.NewTenantHandler(deps.TenantService)
	_ = tenantHandler // placeholder until proto is generated
	// tenantpb.RegisterTenantServiceServer(server, tenantHandler)

	// Register health check service
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Register reflection service (for debugging)
	reflection.Register(server)

	return server
}

// serveGRPC starts the gRPC server
func serveGRPC(server *grpc.Server, port int, log *zap.Logger) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	log.Info("gRPC server listening", zap.Int("port", port))
	return server.Serve(lis)
}

// gracefulShutdown performs graceful shutdown of all servers
func gracefulShutdown(ctx context.Context, httpServer *http.Server, grpcServer *grpc.Server, log *zap.Logger) {
	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	log.Info("Shutting down HTTP server")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Shutdown gRPC server
	log.Info("Shutting down gRPC server")
	grpcServer.GracefulStop()

	// Wait for context timeout or completion
	<-shutdownCtx.Done()
	if shutdownCtx.Err() == context.DeadlineExceeded {
		log.Warn("Shutdown timeout exceeded, forcing stop")
	}
}

// grpcUnaryInterceptor adds logging and error handling to unary RPCs
func grpcUnaryInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Call handler
		resp, err := handler(ctx, req)

		// Log request
		log.Info("gRPC request",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)

		return resp, err
	}
}

// grpcStreamInterceptor adds logging to stream RPCs
func grpcStreamInterceptor(log *zap.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// Call handler
		err := handler(srv, ss)

		// Log request
		log.Info("gRPC stream",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)

		return err
	}
}

// maskPassword masks passwords in URLs for logging
func maskPassword(url string) string {
	// Simple masking - in production use proper URL parsing
	return url // TODO: implement proper password masking
}
