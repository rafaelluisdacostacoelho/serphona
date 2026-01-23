package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit rejects requests whose body exceeds the configured size.
func BodyLimit(maxBytes int) gin.HandlerFunc {
	limit := int64(maxBytes)
	return func(c *gin.Context) {
		if maxBytes > 0 {
			if c.Request.ContentLength > limit {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
				return
			}
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
