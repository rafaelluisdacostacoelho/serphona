package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements token bucket algorithm for rate limiting
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     int           // requests per minute
	burst    int           // max burst size
	cleanup  time.Duration // cleanup interval
}

type visitor struct {
	tokens     int
	lastAccess time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: requests per minute
// burst: maximum burst size
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		burst:    burst,
		cleanup:  5 * time.Minute,
	}

	// Start cleanup goroutine
	go rl.cleanupVisitors()

	return rl
}

// Middleware returns a Gin middleware handler
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get identifier (IP, tenant_id, or user_id)
		identifier := rl.getIdentifier(c)

		if !rl.allow(identifier) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// allow checks if a request should be allowed
func (rl *RateLimiter) allow(identifier string) bool {
	rl.mu.Lock()
	v, exists := rl.visitors[identifier]
	if !exists {
		v = &visitor{
			tokens:     rl.burst,
			lastAccess: time.Now(),
		}
		rl.visitors[identifier] = v
	}
	rl.mu.Unlock()

	v.mu.Lock()
	defer v.mu.Unlock()

	// Refill tokens based on time passed
	now := time.Now()
	elapsed := now.Sub(v.lastAccess)
	v.lastAccess = now

	// Calculate tokens to add (rate per minute)
	tokensToAdd := int(elapsed.Minutes() * float64(rl.rate))
	v.tokens += tokensToAdd
	if v.tokens > rl.burst {
		v.tokens = rl.burst
	}

	// Check if request can proceed
	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}

// getIdentifier extracts identifier from request
// Priority: tenant_id > user_id > IP address
func (rl *RateLimiter) getIdentifier(c *gin.Context) string {
	// Check query params first
	if tenantID := c.Query("tenant_id"); tenantID != "" {
		return "tenant:" + tenantID
	}

	// Check headers
	if tenantID := c.GetHeader("X-Tenant-ID"); tenantID != "" {
		return "tenant:" + tenantID
	}

	if userID := c.GetHeader("X-User-ID"); userID != "" {
		return "user:" + userID
	}

	// Fallback to IP
	return "ip:" + c.ClientIP()
}

// cleanupVisitors removes inactive visitors
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for id, v := range rl.visitors {
			v.mu.Lock()
			if time.Since(v.lastAccess) > rl.cleanup {
				delete(rl.visitors, id)
			}
			v.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// GetStats returns rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"total_visitors": len(rl.visitors),
		"rate":           rl.rate,
		"burst":          rl.burst,
	}
}
