package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	rate       int           // requests per window
	window     time.Duration // time window
	buckets    map[string]*bucket
	mu         sync.RWMutex
	cleanupTTL time.Duration
}

type bucket struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		rate:       rate,
		window:     window,
		buckets:    make(map[string]*bucket),
		cleanupTTL: window * 10, // cleanup old buckets after 10 windows
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.RLock()
	b, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		b, exists = rl.buckets[key]
		if !exists {
			b = &bucket{
				tokens:     rl.rate,
				lastRefill: time.Now(),
			}
			rl.buckets[key] = b
		}
		rl.mu.Unlock()
	}

	return b.consume(rl.rate, rl.window)
}

// consume attempts to consume a token from the bucket
func (b *bucket) consume(rate int, window time.Duration) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill)

	// Refill tokens based on elapsed time
	if elapsed >= window {
		b.tokens = rate
		b.lastRefill = now
	}

	// Try to consume a token
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// cleanup removes old buckets
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupTTL)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.buckets {
			b.mu.Lock()
			if now.Sub(b.lastRefill) > rl.cleanupTTL {
				delete(rl.buckets, key)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates a Gin middleware for rate limiting
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use IP address as key (could also use API key, user ID, etc.)
		key := c.ClientIP()

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByTenantMiddleware creates rate limiting by tenant ID
func RateLimitByTenantMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract tenant ID from header or query
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			tenantID = c.Query("tenant_id")
		}

		if tenantID == "" {
			// Fall back to IP-based rate limiting
			tenantID = c.ClientIP()
		}

		if !limiter.Allow(tenantID) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Tenant rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByUserMiddleware creates rate limiting by user ID
func RateLimitByUserMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID from context (set by auth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			// Fall back to IP-based rate limiting
			userID = c.ClientIP()
		}

		key := userID.(string)
		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "User rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
