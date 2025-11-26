// ==============================================================================
// Billing Service
// ==============================================================================
// Integrates Stripe (products, plans, subscriptions, invoices).
// Relates tenant_id with Stripe customer_id.
// Exposes: Webhooks Stripe, internal APIs for frontend and tenant-manager.

package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("Starting Billing Service...")

	router := setupRouter()

	srv := &http.Server{
		Addr:         getEnv("HTTP_ADDR", ":8083"),
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

func setupRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "billing-service"})
	})

	// Stripe webhook (raw body needed)
	router.POST("/webhooks/stripe", handleStripeWebhook)

	v1 := router.Group("/api/v1")
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
	// TODO: Create Stripe customer and link to tenant_id
	c.JSON(http.StatusCreated, gin.H{
		"customer_id": "cus_placeholder",
		"tenant_id":   "placeholder",
	})
}

func getCustomer(c *gin.Context) {
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
	c.JSON(http.StatusOK, gin.H{
		"subscriptions": []gin.H{},
	})
}

func createSubscription(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"subscription_id": "sub_placeholder",
	})
}

func getSubscription(c *gin.Context) {
	subID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"subscription_id": subID,
		"status":          "active",
		"plan":            "pro",
	})
}

func updateSubscription(c *gin.Context) {
	subID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"subscription_id": subID,
		"message":         "Subscription updated",
	})
}

func cancelSubscription(c *gin.Context) {
	subID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"subscription_id": subID,
		"message":         "Subscription cancelled",
	})
}

// ==============================================================================
// Invoice Handlers
// ==============================================================================

func listInvoices(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"invoices": []gin.H{},
	})
}

func getInvoice(c *gin.Context) {
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
	c.JSON(http.StatusOK, gin.H{
		"calls":      0,
		"tokens":     0,
		"storage_mb": 0,
		"agents":     0,
	})
}

func createPortalSession(c *gin.Context) {
	// TODO: Create Stripe Billing Portal session
	c.JSON(http.StatusOK, gin.H{
		"url": "https://billing.stripe.com/session/placeholder",
	})
}

func createCheckoutSession(c *gin.Context) {
	// TODO: Create Stripe Checkout session
	c.JSON(http.StatusOK, gin.H{
		"url": "https://checkout.stripe.com/session/placeholder",
	})
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
