// ==============================================================================
// Agent Orchestrator Service
// ==============================================================================
// Orchestrates the cloud of LLM agents, handles routing, delegation,
// and session context management.

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
)

func main() {
	// Initialize logger
	log.Println("Starting Agent Orchestrator Service...")

	// Setup router
	router := setupRouter()

	// Server configuration
	srv := &http.Server{
		Addr:         getEnv("HTTP_ADDR", ":8080"),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
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

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "agent-orchestrator"})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Session management
		sessions := v1.Group("/sessions")
		{
			sessions.POST("", createSession)
			sessions.GET("/:id", getSession)
			sessions.DELETE("/:id", endSession)
			sessions.POST("/:id/messages", sendMessage)
		}

		// Agent routing
		agents := v1.Group("/agents")
		{
			agents.POST("/:id/invoke", invokeAgent)
			agents.POST("/:id/delegate", delegateToAgent)
		}
	}

	return router
}

// ==============================================================================
// Handlers
// ==============================================================================

func createSession(c *gin.Context) {
	// TODO: Implement session creation
	// - Generate session ID
	// - Initialize context in Redis
	// - Return session info
	c.JSON(http.StatusCreated, gin.H{
		"session_id": "sess_placeholder",
		"created_at": time.Now().UTC(),
	})
}

func getSession(c *gin.Context) {
	sessionID := c.Param("id")
	// TODO: Fetch session from Redis
	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"status":     "active",
	})
}

func endSession(c *gin.Context) {
	sessionID := c.Param("id")
	// TODO: Clean up session from Redis
	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"status":     "ended",
	})
}

func sendMessage(c *gin.Context) {
	sessionID := c.Param("id")
	// TODO: Process message through agent pipeline
	// - Route to appropriate agent
	// - Manage context
	// - Emit events to Kafka
	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"response":   "Message processed",
	})
}

func invokeAgent(c *gin.Context) {
	agentID := c.Param("id")
	// TODO: Invoke specific agent
	c.JSON(http.StatusOK, gin.H{
		"agent_id": agentID,
		"result":   "Agent invoked",
	})
}

func delegateToAgent(c *gin.Context) {
	agentID := c.Param("id")
	// TODO: Delegate task to another agent
	c.JSON(http.StatusOK, gin.H{
		"agent_id":  agentID,
		"delegated": true,
	})
}

// ==============================================================================
// Helpers
// ==============================================================================

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
