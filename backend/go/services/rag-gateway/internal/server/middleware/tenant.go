package middleware

import (
	"github.com/gin-gonic/gin"
)

// InjectTenant adds tenant_id into the context from header or fallback value.
func InjectTenant(headerName, fallback string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenant := c.GetHeader(headerName)
		if tenant == "" {
			tenant = fallback
		}

		c.Set("tenant_id", tenant)
		c.Next()
	}
}
