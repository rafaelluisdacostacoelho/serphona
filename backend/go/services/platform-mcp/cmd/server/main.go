package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/platform-mcp/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
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
	authmw.SetAuthMetricsService(serviceName)
	shutdownTracer, err := initTracer(cfg)
	if err != nil {
		log.Fatal("failed to init tracer", zap.Error(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTracer(ctx)
	}()

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
		AllowedAlgs:     cfg.Auth.AllowedAlgs,
		Issuer:          cfg.Auth.Issuer,
		Audience:        cfg.Auth.Audience,
		ServiceAudience: cfg.Auth.ServiceAudience,
		ClockSkew:       cfg.Auth.ClockSkew,
		MaxTokenBytes:   cfg.Auth.MaxTokenBytes,
		JWKSURL:         cfg.Auth.JWKSURL,
		JWKSCacheTTL:    cfg.Auth.JWKSCacheTTL,
		AllowedKIDs:     cfg.Auth.AllowedKIDs,
		RequiredScopes:  cfg.Auth.RequiredScopes,
	})
	if cfg.Auth.JWTSecret != "" {
		authjwt.SetSecret(cfg.Auth.JWTSecret)
	}
}

func buildHTTPServer(cfg *config.Config) *http.Server {
	r := gin.New()
	r.Use(gin.Recovery())

	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "platform_mcp_http_requests_total",
			Help: "HTTP requests processed by platform-mcp",
		},
		[]string{"path", "method", "status", "tenant"},
	)
	requestLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "platform_mcp_http_request_duration_seconds",
			Help:    "HTTP request latency by path/method",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method", "tenant"},
	)
	prometheus.MustRegister(requestCounter, requestLatency)

	// Swagger UI (UI under /swagger/index.html, spec served from /swagger-docs/doc.json to avoid wildcard conflicts)
	r.GET("/swagger-docs/doc.json", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger-docs/doc.json")))

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
	api.Use(authmw.RequireAuth(), tenantGuard(), traceMiddleware(), metricsMiddleware(requestCounter, requestLatency))
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

// tenantGuard enforces tenant consistency between claims and headers.
func tenantGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := authmw.GetClaimsFromContext(c)
		if err != nil {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "missing auth claims", nil)
			c.Abort()
			return
		}

		tenantID := claims.TenantID
		if tenantID == "" {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusUnauthorized, "unauthorized", "tenant_id missing in token", nil)
			c.Abort()
			return
		}

		headerTenant := c.GetHeader(authmw.TenantIDHeader)
		if headerTenant == "" {
			c.Request.Header.Set(authmw.TenantIDHeader, tenantID)
		} else if tenantID != "platform" && headerTenant != tenantID {
			response.WriteError(c.Request.Context(), c.Writer, http.StatusForbidden, "tenant_mismatch", "tenant header does not match token", nil)
			c.Abort()
			return
		}

		c.Request = c.Request.WithContext(authmw.WithTenantID(c.Request.Context(), tenantID))
		c.Next()
	}
}

// traceMiddleware starts a server span and tags it with auth/tool metadata when available.
func traceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		spanName := c.FullPath()
		if spanName == "" {
			spanName = c.Request.Method + " " + c.Request.URL.Path
		}

		ctx, span := otel.Tracer("platform-mcp/http").Start(c.Request.Context(), spanName, trace.WithSpanKind(trace.SpanKindServer))

		attrs := []attribute.KeyValue{
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.path", c.FullPath()),
		}

		if claims, err := authmw.GetClaimsFromContext(c); err == nil {
			attrs = append(attrs,
				attribute.String("tenant.id", claims.TenantID),
				attribute.String("user.id", claims.UserID),
				attribute.String("service", claims.Service),
			)
		}

		span.SetAttributes(attrs...)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		span.SetAttributes(attribute.Int("http.status_code", c.Writer.Status()))
		span.End()
	}
}

// metricsMiddleware records HTTP request totals and latency with tenant labels when available.
func metricsMiddleware(counter *prometheus.CounterVec, histogram *prometheus.HistogramVec) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := fmt.Sprintf("%d", c.Writer.Status())
		tenant := "unknown"
		if claims, err := authmw.GetClaimsFromContext(c); err == nil && claims.TenantID != "" {
			tenant = claims.TenantID
		}
		counter.WithLabelValues(c.FullPath(), c.Request.Method, status, tenant).Inc()
		histogram.WithLabelValues(c.FullPath(), c.Request.Method, tenant).Observe(time.Since(start).Seconds())
	}
}

func buildGRPCServer(cfg *config.Config) *grpc.Server {
	maxRecv := cfg.GRPC.MaxRecvMsgSizeMB * 1024 * 1024
	maxSend := cfg.GRPC.MaxSendMsgSizeMB * 1024 * 1024

	grpcRequestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "platform_mcp_grpc_requests_total",
			Help: "gRPC requests processed by platform-mcp",
		},
		[]string{"method", "code", "tenant"},
	)
	grpcRequestLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "platform_mcp_grpc_request_duration_seconds",
			Help:    "gRPC request latency by method",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "tenant"},
	)
	prometheus.MustRegister(grpcRequestCounter, grpcRequestLatency)

	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(authmw.UnaryAuthInterceptor(), tenantUnaryGuard(), grpcUnaryMetrics(grpcRequestCounter, grpcRequestLatency)),
		grpc.ChainStreamInterceptor(authmw.StreamAuthInterceptor(), tenantStreamGuard(), grpcStreamMetrics(grpcRequestCounter, grpcRequestLatency)),
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

// tenantUnaryGuard enforces tenant consistency for unary RPCs and sets tenant in context.
func tenantUnaryGuard() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		claims, err := authmw.ClaimsFromContext(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "missing auth claims")
		}

		tenantID := claims.TenantID
		if tenantID == "" {
			return nil, status.Error(codes.Unauthenticated, "tenant_id missing in token")
		}

		headerTenant := tenantFromMetadata(ctx)
		if headerTenant != "" && tenantID != "platform" && headerTenant != tenantID {
			return nil, status.Error(codes.PermissionDenied, "tenant header does not match token")
		}

		ctx = authmw.WithTenantID(ctx, tenantID)
		return handler(ctx, req)
	}
}

func grpcUnaryMetrics(counter *prometheus.CounterVec, histogram *prometheus.HistogramVec) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err).String()
		tenant := tenantFromContext(ctx)
		counter.WithLabelValues(info.FullMethod, code, tenant).Inc()
		histogram.WithLabelValues(info.FullMethod, tenant).Observe(time.Since(start).Seconds())
		return resp, err
	}
}

// tenantStreamGuard enforces tenant consistency for stream RPCs and sets tenant in context.
func tenantStreamGuard() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		claims, err := authmw.ClaimsFromContext(ctx)
		if err != nil {
			return status.Error(codes.Unauthenticated, "missing auth claims")
		}

		tenantID := claims.TenantID
		if tenantID == "" {
			return status.Error(codes.Unauthenticated, "tenant_id missing in token")
		}

		headerTenant := tenantFromMetadata(ctx)
		if headerTenant != "" && tenantID != "platform" && headerTenant != tenantID {
			return status.Error(codes.PermissionDenied, "tenant header does not match token")
		}

		ctx = authmw.WithTenantID(ctx, tenantID)
		return handler(srv, &tenantWrappedStream{ServerStream: ss, ctx: ctx})
	}
}

func grpcStreamMetrics(counter *prometheus.CounterVec, histogram *prometheus.HistogramVec) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, ss)
		code := status.Code(err).String()
		tenant := tenantFromContext(ss.Context())
		counter.WithLabelValues(info.FullMethod, code, tenant).Inc()
		histogram.WithLabelValues(info.FullMethod, tenant).Observe(time.Since(start).Seconds())
		return err
	}
}

func tenantFromMetadata(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(strings.ToLower(authmw.TenantIDHeader)); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}
	return ""
}

func tenantFromContext(ctx context.Context) string {
	tenant, err := authmw.TenantIDFromContext(ctx)
	if err != nil || tenant == "" {
		return "unknown"
	}
	return tenant
}

type tenantWrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *tenantWrappedStream) Context() context.Context {
	return w.ctx
}

func initTracer(cfg *config.Config) (func(context.Context) error, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(propagator)

	if !cfg.Tracing.Enabled {
		tp := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(tp)
		return tp.Shutdown, nil
	}

	clientOpts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Tracing.Endpoint)}
	if cfg.Tracing.Insecure {
		clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, clientOpts...)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.version", serviceVersion),
			attribute.String("deployment.environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.Tracing.SampleRate))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
