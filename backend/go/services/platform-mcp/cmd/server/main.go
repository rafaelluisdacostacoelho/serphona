package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"platform-mcp/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	serviceName    = "platform-mcp"
	serviceVersion = "0.1.0"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := newLogger(cfg.Environment)
	defer log.Sync()

	log.Info("Starting service", zap.String("service", serviceName), zap.String("version", serviceVersion), zap.String("env", cfg.Environment))

	configureAuth(cfg)

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	httpServer := buildHTTPServer(cfg)
	grpcServer := buildGRPCServer(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatal("gRPC listen failed", zap.Error(err), zap.String("addr", addr))
		}
		log.Info("gRPC listening", zap.String("addr", addr))
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("gRPC server stopped", zap.Error(err))
			stop()
		}
	}()

	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port)
		log.Info("HTTP listening", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server stopped", zap.Error(err))
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	log.Info("Shutting down HTTP")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Warn("HTTP shutdown error", zap.Error(err))
	}

	log.Info("Shutting down gRPC")
	grpcServer.GracefulStop()

	log.Info("Service stopped")
}

func newLogger(env string) *zap.Logger {
	if env == "production" {
		log, _ := zap.NewProduction()
		return log
	}
	log, _ := zap.NewDevelopment()
	return log
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

func buildHTTPServer(cfg *config.Config) *http.Server {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{"status": "ok"})
	})
	if cfg.HTTP.ReadTimeout > 0 {
		r.GET("/ready", func(c *gin.Context) {
			response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{"status": "ready"})
		})
	}

	if cfg.Metrics.Enabled {
		r.GET(cfg.Metrics.Path, gin.WrapH(promhttp.Handler()))
	}

	if cfg.Pprof.Enabled {
		pp := r.Group("/debug/pprof")
		pp.GET("/", gin.WrapF(pprof.Index))
		pp.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		pp.GET("/profile", gin.WrapF(pprof.Profile))
		pp.GET("/symbol", gin.WrapF(pprof.Symbol))
		pp.POST("/symbol", gin.WrapF(pprof.Symbol))
		pp.GET("/trace", gin.WrapF(pprof.Trace))
		pp.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		pp.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		pp.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		pp.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		pp.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}

	api := r.Group("/api/v1")
	api.Use(authmw.RequireAuth())
	api.GET("/ping", func(c *gin.Context) {
		response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{"message": "pong", "service": serviceName})
	})

	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:           r,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		MaxHeaderBytes:    cfg.HTTP.MaxHeaderBytes,
	}
}

func buildGRPCServer(cfg *config.Config) *grpc.Server {
	maxRecv := cfg.GRPC.MaxRecvMsgSizeMB * 1024 * 1024
	maxSend := cfg.GRPC.MaxSendMsgSizeMB * 1024 * 1024

	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(authmw.UnaryAuthInterceptor()),
		grpc.ChainStreamInterceptor(authmw.StreamAuthInterceptor()),
		grpc.MaxRecvMsgSize(maxRecv),
		grpc.MaxSendMsgSize(maxSend),
		grpc.ConnectionTimeout(cfg.GRPC.ConnectionTimeout),
	}

	server := grpc.NewServer(opts...)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	if cfg.GRPC.ReflectionEnabled && cfg.Environment != "production" {
		reflection.Register(server)
	}

	return server
}
