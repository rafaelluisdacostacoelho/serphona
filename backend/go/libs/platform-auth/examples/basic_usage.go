package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/serphona/serphona/backend/go/libs/platform-auth/client"
	authjwt "github.com/serphona/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/serphona/serphona/backend/go/libs/platform-auth/middleware"
)

func main() {
	authjwt.MustSetSecretFromEnv()

	authClient, err := client.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	protected := router.Group("/api/v1")
	protected.Use(middleware.RequireAuth())
	{
		protected.GET("/profile", func(c *gin.Context) {
			claims, _ := middleware.GetClaimsFromContext(c)

			c.JSON(200, gin.H{
				"message": "Perfil do usuario",
				"user": gin.H{
					"id":       claims.UserID,
					"email":    claims.Email,
					"name":     claims.Name,
					"role":     claims.Role,
					"tenantId": claims.TenantID,
				},
			})
		})

		protected.GET("/data", getData)
	}

	admin := router.Group("/api/v1/admin")
	admin.Use(middleware.RequireAuth())
	admin.Use(middleware.RequireAdmin())
	{
		admin.GET("/users", listUsers)
		admin.GET("/reports", getAdminReports)
	}

	superadmin := router.Group("/api/v1/superadmin")
	superadmin.Use(middleware.RequireAuth())
	superadmin.Use(middleware.RequireSuperAdmin())
	{
		superadmin.GET("/system", getSystemInfo)
	}

	router.GET("/api/v1/validate-demo", func(c *gin.Context) {
		token := c.GetHeader("Authorization")

		claims, err := authjwt.ValidateTokenFromHeader(token)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			return
		}

		tokenStr, _ := authjwt.ExtractTokenFromHeader(token)
		claimsFromGateway, err := authClient.ValidateToken(tokenStr)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"local":   claims,
			"gateway": claimsFromGateway,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	fmt.Printf("Servidor rodando na porta %s\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func getData(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	tenantID, _ := middleware.GetTenantIDFromContext(c)

	c.JSON(200, gin.H{
		"message":  "Dados do tenant",
		"userId":   userID,
		"tenantId": tenantID,
		"data": []gin.H{
			{"id": 1, "name": "Item 1"},
			{"id": 2, "name": "Item 2"},
		},
	})
}

func listUsers(c *gin.Context) {
	claims, _ := middleware.GetClaimsFromContext(c)

	c.JSON(200, gin.H{
		"message": "Lista de usuarios (somente admin)",
		"admin":   claims.Email,
		"users": []gin.H{
			{"id": 1, "name": "User 1"},
			{"id": 2, "name": "User 2"},
		},
	})
}

func getAdminReports(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Relatorios administrativos",
		"reports": []string{"Report 1", "Report 2"},
	})
}

func getSystemInfo(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Informacoes do sistema (somente superadmin)",
		"info": gin.H{
			"version": "1.0.0",
			"status":  "running",
		},
	})
}
