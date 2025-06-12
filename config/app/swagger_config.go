package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BasicAuthMiddleware handles basic authentication for Swagger UI
func BasicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Get the Basic Auth credentials
		username, password, ok := c.Request.BasicAuth()
		if !ok || !checkCredentials(username, password) {
			c.Header("WWW-Authenticate", `Basic realm="Swagger UI"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

// checkCredentials validates the username and password
func checkCredentials(username, password string) bool {
	// Replace these with your desired credentials
	// In production, you should use environment variables or a secure configuration
	return username == "j_ai_trade_admin" && password == "Jnguyen123456@"
}
