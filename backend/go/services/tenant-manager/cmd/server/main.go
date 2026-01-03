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
	apikeyapp "tenant-manager/internal/application/apikey"
	"tenant-manager/internal/application/tenant"
	"tenant-manager/internal/config"
	domainapikey "tenant-manager/internal/domain/apikey"
	tenantDomain "tenant-manager/internal/domain/tenant"
	"tenant-manager/pkg/logger"
	tenantpb "tenant-manager/proto"

	"github.com/gin-gonic/gin"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

const (
	serviceVersion   = "1.0.0"
	readTenantScope  = "read:tenants"
	writeTenantScope = "write:tenants"
)

var (
	tenantRateMetrics *middleware.TenantRateMetrics
	tenantRateLimiter *middleware.TenantRateLimiter
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

	configureAuth(cfg)
	tenantRateMetrics = middleware.NewTenantRateMetrics()
	tenantRateLimiter = middleware.NewTenantRateLimiter(cfg.Server.TenantRateLimitRPM, tenantRateMetrics)

	log.Info("Starting Tenant Manager Service",
		zap.String("service", cfg.Service.Name),
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
		if err := serveGRPC(grpcServer, cfg.GRPC.Host, cfg.GRPC.Port, log); err != nil {
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
	gracefulShutdown(cfg, httpServer, grpcServer, log)
	log.Info("Service stopped gracefully")
}

// Dependencies holds all service dependencies
type Dependencies struct {
	// Repositories
	TenantRepo tenantDomain.Repository
	APIKeyRepo domainapikey.Repository

	// Services
	TenantService *tenant.Service
	APIKeyService *apikeyapp.Service

	// Infrastructure
	DB             *postgres.DB
	Cache          *redis.Cache
	EventPublisher tenantDomain.EventPublisher
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
		log.Warn("Failed to connect to Redis, using no-op cache", zap.Error(err))
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
		log.Warn("Failed to connect to Kafka, using no-op publisher", zap.Error(err))
		producer = nil
	} else {
		deps.Producer = producer
		deps.EventPublisher = kafka.NewEventPublisher(producer, cfg.Kafka.TopicPrefix, cfg.Kafka.DLQTopic)
		cleanupFuncs = append(cleanupFuncs, func() {
			log.Info("Closing Kafka connection")
			if err := producer.Close(); err != nil {
				log.Error("Error closing Kafka", zap.Error(err))
			}
		})
	}
	if deps.EventPublisher == nil {
		deps.EventPublisher = kafka.NewNoopPublisher()
	}

	// Initialize repositories
	deps.TenantRepo = postgres.NewTenantRepository(db)
	deps.APIKeyRepo = postgres.NewAPIKeyRepository(db)

	// Initialize domain services
	tenantDomainService := tenantDomain.NewService(deps.TenantRepo)
	apiKeyDomainService := domainapikey.NewService(deps.APIKeyRepo)

	// Initialize application services
	var tenantCache tenantDomain.Cache
	var apiKeyCache apikeyapp.Cache
	if redisCache != nil {
		tenantCache = redis.NewTenantCache(redisCache, cfg.Redis.CacheTTL)
		apiKeyCache = redisCache
	} else {
		tenantCache = redis.NoopCache{}
	}

	deps.TenantService = tenant.NewService(
		deps.TenantRepo,
		nil, // tenant service no longer depends on api key repo directly; API keys handled by dedicated service
		tenantCache,
		deps.EventPublisher,
		log,
	)
	deps.APIKeyService = apikeyapp.NewService(
		apiKeyDomainService,
		deps.TenantRepo,
		nil, // TODO: wire event publisher
		apiKeyCache,
	)
	_ = tenantDomainService // Use domain service when needed

	// Optionally start metrics server on dedicated port
	if cfg.Metrics.Enabled {
		go startMetricsServer(cfg.Metrics, log)
	}

	cleanup := func() {
		for i := len(cleanupFuncs) - 1; i >= 0; i-- {
			cleanupFuncs[i]()
		}
	}

	log.Info("Dependencies initialized successfully")
	return deps, cleanup, nil
}

func configureAuth(cfg *config.Config) {
	authjwt.SetSecret(cfg.JWT.Secret)
	vc := authjwt.ValidationConfig{Issuer: cfg.JWT.Issuer}
	if len(cfg.JWT.Audience) > 0 {
		vc.Audience = cfg.JWT.Audience[0]
	}
	if cfg.JWT.PublicKey != "" {
		vc.AllowedAlgs = []string{"HS256", "RS256"}
	}
	authjwt.SetValidationConfig(vc)
}

// startHTTPServer creates and configures the HTTP server
func startHTTPServer(cfg *config.Config, deps *Dependencies, log *zap.Logger) *http.Server {
	// Create Gin router
	r := gin.New()

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.ZapLogger(log))
	r.Use(middleware.BodyLimit(cfg.Server.BodyLimitBytes))
	r.Use(middleware.RateLimit(cfg.Server.RateLimitRPM))
	r.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowedOrigins:   cfg.Server.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           10 * time.Minute,
	}))

	// Health checks (no auth required)
	healthHandler := httpHandler.NewHealthHandler(deps.DB.Pool, nil)
	r.GET("/health", func(c *gin.Context) {
		healthHandler.Health(c.Writer, c.Request)
	})
	r.GET("/ready", func(c *gin.Context) {
		healthHandler.Ready(c.Writer, c.Request)
	})

	// Metrics endpoint (Prometheus exposition)
	r.GET("/metrics", func(c *gin.Context) {
		metricsHandler(c.Writer, c.Request)
	})

	// Serve OpenAPI/Swagger UI (static)
	r.Static("/docs", "./api/openapi")

	// API routes with authentication
	api := r.Group("/api/v1")
	api.Use(middleware.JWTAuth(cfg.JWT.Secret, cfg.JWT.PublicKey, cfg.JWT.Issuer, cfg.JWT.Audience))
	api.Use(middleware.TenantRateLimitWithLimiter(tenantRateLimiter))
	{
		// Tenant routes
		tenantHandler := httpHandler.NewGinTenantHandler(deps.TenantService, log)
		api.POST("/tenants", middleware.RequireScopes(writeTenantScope), tenantHandler.Create)
		api.GET("/tenants", middleware.RequireScopes(readTenantScope), tenantHandler.List)
		api.GET("/tenants/:id", middleware.RequireScopes(readTenantScope), tenantHandler.Get)
		api.PUT("/tenants/:id", middleware.RequireScopes(writeTenantScope), tenantHandler.Update)
		api.DELETE("/tenants/:id", middleware.RequireScopes(writeTenantScope), tenantHandler.Delete)
		api.GET("/tenants/:id/quota", middleware.RequireScopes(readTenantScope), tenantHandler.GetQuota)
		api.PUT("/tenants/:id/quota", middleware.RequireScopes(writeTenantScope), tenantHandler.UpdateQuota)
		api.POST("/tenants/:id/usage", middleware.RequireScopes(writeTenantScope), tenantHandler.IncrementUsage)

		// API Key routes
		apiKeyHandler := httpHandler.NewGinAPIKeyHandler(deps.APIKeyService, log)
		api.POST("/tenants/:id/api-keys", middleware.RequireScopes(writeTenantScope), apiKeyHandler.Create)
		api.GET("/tenants/:id/api-keys", middleware.RequireScopes(readTenantScope), apiKeyHandler.List)
		api.DELETE("/tenants/:id/api-keys/:keyId", middleware.RequireScopes(writeTenantScope), apiKeyHandler.Delete)
	}

	// Create HTTP server
	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           r,
		ReadTimeout:       cfg.Server.ReadTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
	}

	return server
}

// startGRPCServer creates and configures the gRPC server
func startGRPCServer(cfg *config.Config, deps *Dependencies, log *zap.Logger) *grpc.Server {
	grpcCfg := cfg.GRPC

	maxRecv := grpcCfg.MaxRecvMsgSizeMB * 1024 * 1024
	maxSend := grpcCfg.MaxSendMsgSizeMB * 1024 * 1024

	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			authmw.UnaryAuthInterceptor(),
			middleware.TenantUnaryRateLimit(tenantRateLimiter),
			enforceDeadline(cfg.GRPC.DefaultRequestTimeout, log),
			grpcUnaryInterceptor(log),
		),
		grpc.ChainStreamInterceptor(
			authmw.StreamAuthInterceptor(),
			middleware.TenantStreamRateLimit(tenantRateLimiter),
			grpcStreamInterceptor(log),
		),
		grpc.MaxRecvMsgSize(maxRecv),
		grpc.MaxSendMsgSize(maxSend),
		grpc.ConnectionTimeout(grpcCfg.ConnectionTimeout),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     grpcCfg.Keepalive.MaxConnectionIdle,
			MaxConnectionAge:      grpcCfg.Keepalive.MaxConnectionAge,
			MaxConnectionAgeGrace: grpcCfg.Keepalive.MaxConnectionAgeGrace,
			Time:                  grpcCfg.Keepalive.Time,
			Timeout:               grpcCfg.Keepalive.Timeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcCfg.Keepalive.MinTime,
			PermitWithoutStream: grpcCfg.Keepalive.PermitWithoutStream,
		}),
	}

	server := grpc.NewServer(opts...)

	tenantHandler := handler.NewTenantHandler(deps.TenantService)
	tenantpb.RegisterTenantServiceServer(server, tenantHandler)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Reflection: preferir desabilitado em produção
	if grpcCfg.ReflectionEnabled && cfg.Environment != "production" {
		reflection.Register(server)
	}

	return server
}

// serveGRPC starts the gRPC server
func serveGRPC(server *grpc.Server, host string, port int, log *zap.Logger) error {
	addr := fmt.Sprintf("%s:%d", host, port)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	log.Info("gRPC server listening", zap.String("address", addr))
	return server.Serve(lis)
}

// gracefulShutdown performs graceful shutdown of all servers
func gracefulShutdown(cfg *config.Config, httpServer *http.Server, grpcServer *grpc.Server, log *zap.Logger) {
	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
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

func enforceDeadline(defaultTimeout time.Duration, log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
			defer cancel()
		}
		return handler(ctx, req)
	}
}

// maskPassword masks passwords in URLs for logging
func maskPassword(url string) string {
	// Simple masking - in production use proper URL parsing
	return url // TODO: implement proper password masking
}

// startMetricsServer runs a minimal metrics endpoint on a dedicated port if enabled.
func startMetricsServer(cfg config.MetricsConfig, log *zap.Logger) {
	addr := fmt.Sprintf(":%d", cfg.Port)
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", metricsHandler)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("Metrics server listening", zap.String("address", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Warn("Metrics server error", zap.Error(err))
		}
	}()
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP tenant_manager_up Service health\n# TYPE tenant_manager_up gauge\ntenant_manager_up 1\n")

	if tenantRateMetrics != nil {
		snapshot := tenantRateMetrics.Snapshot()
		fmt.Fprintf(w, "# HELP tenant_manager_tenant_rate_allowed Total requests allowed per tenant\n")
		fmt.Fprintf(w, "# TYPE tenant_manager_tenant_rate_allowed counter\n")
		for tenantID, s := range snapshot {
			fmt.Fprintf(w, "tenant_manager_tenant_rate_allowed{tenant_id=\"%s\"} %d\n", tenantID, s.Allowed)
		}
		fmt.Fprintf(w, "# HELP tenant_manager_tenant_rate_blocked Total requests blocked per tenant\n")
		fmt.Fprintf(w, "# TYPE tenant_manager_tenant_rate_blocked counter\n")
		for tenantID, s := range snapshot {
			fmt.Fprintf(w, "tenant_manager_tenant_rate_blocked{tenant_id=\"%s\"} %d\n", tenantID, s.Blocked)
		}
	}
}
