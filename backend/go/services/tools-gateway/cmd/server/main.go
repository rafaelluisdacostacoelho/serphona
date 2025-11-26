// ==============================================================================
// Tools Gateway Service
// ==============================================================================
// Manages registration and execution of Tools (external APIs).
// Abstracts GET/POST/PUT/DELETE with input/output schema.
// Per-tenant and per-tool authentication. Logging for analytics.

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
	log.Println("Starting Tools Gateway Service...")

	router := setupRouter()

	srv := &http.Server{
		Addr:         getEnv("HTTP_ADDR", ":8081"),
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
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tools-gateway"})
	})

	v1 := router.Group("/api/v1")
	{
		// Tool management
		tools := v1.Group("/tools")
		{
			tools.GET("", listTools)
			tools.POST("", createTool)
			tools.GET("/:id", getTool)
			tools.PUT("/:id", updateTool)
			tools.DELETE("/:id", deleteTool)
		}

		// Tool execution
		v1.POST("/tools/:id/execute", executeTool)

		// Tool schemas
		v1.GET("/tools/:id/schema", getToolSchema)
	}

	return router
}

// ==============================================================================
// Tool Management Handlers
// ==============================================================================

func listTools(c *gin.Context) {
	// TODO: List tools for tenant (from tenant_id in JWT)
	c.JSON(http.StatusOK, gin.H{
		"tools": []gin.H{},
		"total": 0,
	})
}

func createTool(c *gin.Context) {
	// TODO: Create new tool
	// - Validate schema
	// - Store in DB
	// - Associate with tenant
	c.JSON(http.StatusCreated, gin.H{
		"id":      "tool_placeholder",
		"message": "Tool created",
	})
}

func getTool(c *gin.Context) {
	toolID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     toolID,
		"name":   "Example Tool",
		"method": "GET",
		"url":    "https://api.example.com/endpoint",
	})
}

func updateTool(c *gin.Context) {
	toolID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      toolID,
		"message": "Tool updated",
	})
}

func deleteTool(c *gin.Context) {
	toolID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      toolID,
		"message": "Tool deleted",
	})
}

// ==============================================================================
// Tool Execution Handlers
// ==============================================================================

func executeTool(c *gin.Context) {
	toolID := c.Param("id")
	// TODO: Execute tool
	// - Fetch tool config from DB
	// - Validate input against schema
	// - Make HTTP request
	// - Log execution for analytics (Kafka)
	// - Return response
	c.JSON(http.StatusOK, gin.H{
		"tool_id":    toolID,
		"status":     "executed",
		"response":   gin.H{},
		"latency_ms": 0,
	})
}

func getToolSchema(c *gin.Context) {
	toolID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"tool_id": toolID,
		"input_schema": gin.H{
			"type":       "object",
			"properties": gin.H{},
		},
		"output_schema": gin.H{
			"type":       "object",
			"properties": gin.H{},
		},
	})
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
