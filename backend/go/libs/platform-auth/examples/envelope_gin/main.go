package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
)

// Minimal Gin example using response envelopes and auth middleware.
func main() {
	authjwt.MustSetSecretFromEnv()

	r := gin.New()
	r.Use(gin.Recovery(), middleware.GinRecovery())

	api := r.Group("/api")
	api.Use(middleware.RequireAuth())
	{
		api.GET("/ping", handlePing)
		api.GET("/invoices", listInvoices)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	if err := r.Run(":" + port); err != nil {
		panic(err)
	}
}

func handlePing(c *gin.Context) {
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, gin.H{"pong": true})
}

func listInvoices(c *gin.Context) {
	data := []gin.H{{"id": "inv-1", "status": "paid"}}
	meta := response.WithPagination(response.Pagination{Page: 1, PageSize: 10, Total: 1, TotalPages: 1})
	response.WriteSuccess(c.Request.Context(), c.Writer, http.StatusOK, data, meta)
}
