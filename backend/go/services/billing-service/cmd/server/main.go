// ==============================================================================
// Billing Service
// ==============================================================================
// Integrates Stripe (products, plans, subscriptions, invoices).
// Relates tenant_id with Stripe customer_id.
// Exposes: Webhooks Stripe, internal APIs for frontend and tenant-manager.

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	httpmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/adapter/http/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/adapter/kafka"
	pgrepo "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/adapter/postgres"
	walletapp "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/application/wallet"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/config"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.Println("Starting Billing Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	authmw.SetAuthMetricsService(cfg.Service.Name)

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	walletRepo := pgrepo.NewWalletRepository(db)
	configPricingTable := walletapp.NewConfigPricingTable(cfg.Pricing)
	pricingTable := pgrepo.NewPricingTable(db, configPricingTable)
	walletSvc := walletapp.NewService(walletRepo, cfg.Wallet.DefaultCurrency, cfg.Wallet.InitialCredits, pricingTable)

	usageConsumer, err := kafka.NewUsageConsumer(cfg.Kafka, walletSvc, log.Default())
	if err != nil {
		log.Printf("Kafka consumer disabled (init error): %v", err)
	} else {
		usageCtx, usageCancel := context.WithCancel(context.Background())
		usageConsumer.Start(usageCtx)
		defer usageCancel()
		defer usageConsumer.Close()
	}

	// Setup router
	router := setupRouter(cfg, db)

	// Setup server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutMs) * time.Millisecond,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutMs) * time.Millisecond,
	}

	// Start server
	go func() {
		log.Printf("Server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
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

func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(gormpostgres.Open(cfg.Database.URL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	log.Println("Database connection established")
	return db, nil
}

func setupRouter(cfg *config.Config, db *gorm.DB) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(limitBody(cfg.Server.MaxBodyBytes))
	router.Use(cors(cfg.Server))
	router.Use(gin.Logger())

	if cfg.Observability.EnableMetrics {
		httpmw.SetMetricsRegisterer(prometheus.DefaultRegisterer)
		authmw.SetMetricsRegisterer(httpmw.MetricsRegisterer())
		router.Use(httpmw.Metrics(cfg.Service.Name))
		router.Use(httpmw.AuthMetrics(cfg.Service.Name))
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "billing-service"})
	})

	if cfg.Observability.EnableMetrics {
		// Prometheus metrics endpoint
		router.GET("/metrics", gin.WrapH(promhttp.HandlerFor(httpmw.MetricsGatherer(), promhttp.HandlerOpts{})))
	}

	// Stripe webhook (raw body needed)
	router.POST("/webhooks/stripe", handleStripeWebhook)

	v1 := router.Group("/api/v1")
	v1.Use(authmw.RequireAuth())
	{
		// Customers
		customers := v1.Group("/customers")
		{
			customers.POST("", createCustomer)
			customers.GET("/:id", getCustomer)
		}

		// Subscriptions
		subscriptions := v1.Group("/subscriptions")
		{
			subscriptions.GET("", listSubscriptions)
			subscriptions.POST("", createSubscription)
			subscriptions.GET("/:id", getSubscription)
			subscriptions.PUT("/:id", updateSubscription)
			subscriptions.DELETE("/:id", cancelSubscription)
		}

		// Invoices
		invoices := v1.Group("/invoices")
		{
			invoices.GET("", listInvoices)
			invoices.GET("/:id", getInvoice)
		}

		// Plans & Products
		v1.GET("/plans", listPlans)
		v1.GET("/products", listProducts)

		// Usage & Billing Portal
		v1.GET("/usage", getUsage)
		v1.POST("/portal-session", createPortalSession)
		v1.POST("/checkout-session", createCheckoutSession)
	}

	return router
}

// ==============================================================================
// Stripe Webhook Handler
// ==============================================================================

func handleStripeWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
		return
	}

	// TODO: Verify Stripe signature
	// TODO: Process webhook events:
	// - customer.subscription.created
	// - customer.subscription.updated
	// - customer.subscription.deleted
	// - invoice.payment_succeeded
	// - invoice.payment_failed

	log.Printf("Received Stripe webhook: %d bytes", len(body))
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// ==============================================================================
// Customer Handlers
// ==============================================================================

func createCustomer(c *gin.Context) {
	tenantID, ok := tenantFromContext(c)
	if !ok {
		return
	}
	// TODO: Create Stripe customer and link to tenant_id
	c.JSON(http.StatusCreated, gin.H{
		"customer_id": "cus_placeholder",
		"tenant_id":   tenantID,
	})
}

func getCustomer(c *gin.Context) {
	if _, ok := tenantFromContext(c); !ok {
		return
	}
	customerID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"customer_id": customerID,
		"email":       "customer@example.com",
	})
}

// ==============================================================================
// Subscription Handlers
// ==============================================================================

func listSubscriptions(c *gin.Context) {
	if _, ok := tenantFromContext(c); !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"subscriptions": []gin.H{},
	})
}

func createSubscription(c *gin.Context) {
	if _, ok := tenantFromContext(c); !ok {
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"subscription_id": "sub_placeholder",
	})
}

func getSubscription(c *gin.Context) {
	if _, ok := tenantFromContext(c); !ok {
		return
	}
	subID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"subscription_id": subID,
		"status":          "active",
		"plan":            "pro",
	})
}

func updateSubscription(c *gin.Context) {
	if _, ok := tenantFromContext(c); !ok {
		return
	}
	subID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"subscription_id": subID,
		"message":         "Subscription updated",
	})
}

func cancelSubscription(c *gin.Context) {
	if _, ok := tenantFromContext(c); !ok {
		return
	}
	subID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"subscription_id": subID,
		"message":         "Subscription cancelled",
	})
}

// ==============================================================================
// Middleware
// ==============================================================================

func limitBody(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 {
			c.Next()
			return
		}

		limited := http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		buf := &bytes.Buffer{}
		if _, err := buf.ReadFrom(limited); err != nil {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
		c.Next()
	}
}

func cors(cfg config.ServerConfig) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{})
	for _, o := range cfg.AllowedOrigins {
		allowedOrigins[o] = struct{}{}
	}
	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if len(allowedOrigins) > 0 {
				if _, ok := allowedOrigins[origin]; !ok {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			}
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		c.Writer.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ==============================================================================
// Invoice Handlers
// ==============================================================================

func listInvoices(c *gin.Context) {
	if _, ok := tenantFromContext(c); !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"invoices": []gin.H{},
	})
}

func getInvoice(c *gin.Context) {
	if _, ok := tenantFromContext(c); !ok {
		return
	}
	invoiceID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"invoice_id": invoiceID,
		"status":     "paid",
	})
}

// ==============================================================================
// Plans & Products Handlers
// ==============================================================================

func listPlans(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"plans": []gin.H{
			{"id": "free", "name": "Free", "price": 0},
			{"id": "starter", "name": "Starter", "price": 49},
			{"id": "pro", "name": "Pro", "price": 199},
			{"id": "enterprise", "name": "Enterprise", "price": 0},
		},
	})
}

func listProducts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"products": []gin.H{},
	})
}

// ==============================================================================
// Usage & Portal Handlers
// ==============================================================================

func getUsage(c *gin.Context) {
	if _, ok := tenantFromContext(c); !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"calls":      0,
		"tokens":     0,
		"storage_mb": 0,
		"agents":     0,
	})
}

func createPortalSession(c *gin.Context) {
	if _, ok := tenantFromContext(c); !ok {
		return
	}
	// TODO: Create Stripe Billing Portal session
	c.JSON(http.StatusOK, gin.H{
		"url": "https://billing.stripe.com/session/placeholder",
	})
}

func createCheckoutSession(c *gin.Context) {
	if _, ok := tenantFromContext(c); !ok {
		return
	}
	// TODO: Create Stripe Checkout session
	c.JSON(http.StatusOK, gin.H{
		"url": "https://checkout.stripe.com/session/placeholder",
	})
}

func tenantFromContext(c *gin.Context) (string, bool) {
	tenantID, err := authmw.GetTenantIDFromContext(c)
	if err != nil || tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context required"})
		return "", false
	}
	c.Request.Header = authmw.EnsureTenantHeader(c.Request.Header, tenantID)
	c.Request = c.Request.WithContext(authmw.WithTenantID(c.Request.Context(), tenantID))
	return tenantID, true
}
