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
	// Configurar JWT secret (deve ser o mesmo em todos os services)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET não configurado")
	}
	authjwt.SetSecret(jwtSecret)

	// Criar cliente HTTP para auth-gateway
	authGatewayURL := os.Getenv("AUTH_GATEWAY_URL")
	if authGatewayURL == "" {
		authGatewayURL = "http://localhost:8080"
	}
	authClient := client.New(authGatewayURL)

	// Criar router
	router := gin.Default()

	// Rotas públicas
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Rotas protegidas - requer autenticação
	protected := router.Group("/api/v1")
	protected.Use(middleware.RequireAuth())
	{
		// Rota acessível por qualquer usuário autenticado
		protected.GET("/profile", func(c *gin.Context) {
			// Obter informações do usuário do contexto
			claims, _ := middleware.GetClaimsFromContext(c)

			c.JSON(200, gin.H{
				"message": "Perfil do usuário",
				"user": gin.H{
					"id":       claims.UserID,
					"email":    claims.Email,
					"name":     claims.Name,
					"role":     claims.Role,
					"tenantId": claims.TenantID,
				},
			})
		})

		// Rota acessível por qualquer usuário autenticado
		protected.GET("/data", getData)
	}

	// Rotas que requerem role específica
	admin := router.Group("/api/v1/admin")
	admin.Use(middleware.RequireAuth())
	admin.Use(middleware.RequireAdmin())
	{
		admin.GET("/users", listUsers)
		admin.GET("/reports", getAdminReports)
	}

	// Rotas que requerem superadmin
	superadmin := router.Group("/api/v1/superadmin")
	superadmin.Use(middleware.RequireAuth())
	superadmin.Use(middleware.RequireSuperAdmin())
	{
		superadmin.GET("/system", getSystemInfo)
	}

	// Exemplo de uso do cliente HTTP
	router.GET("/api/v1/validate-demo", func(c *gin.Context) {
		token := c.GetHeader("Authorization")

		// Validar token localmente (mais rápido)
		claims, err := authjwt.ValidateTokenFromHeader(token)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			return
		}

		// Ou validar token chamando auth-gateway (mais seguro)
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

	// Iniciar servidor
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
	// Extrair informações do usuário
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
		"message": "Lista de usuários (somente admin)",
		"admin":   claims.Email,
		"users": []gin.H{
			{"id": 1, "name": "User 1"},
			{"id": 2, "name": "User 2"},
		},
	})
}

func getAdminReports(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Relatórios administrativos",
		"reports": []string{"Report 1", "Report 2"},
	})
}

func getSystemInfo(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Informações do sistema (somente superadmin)",
		"info": gin.H{
			"version": "1.0.0",
			"status":  "running",
		},
	})
}
