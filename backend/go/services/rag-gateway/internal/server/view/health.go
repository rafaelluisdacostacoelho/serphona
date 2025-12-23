package view

import "github.com/gin-gonic/gin"

// HealthHandler returns liveness information.
type HealthHandler struct {
	Service string
}

func (h HealthHandler) Get(c *gin.Context) {
	c.JSON(200, gin.H{"status": "healthy", "service": h.Service})
}
