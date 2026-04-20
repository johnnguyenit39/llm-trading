package middlewares

import (
	"j_ai_trade/common"
	"j_ai_trade/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates an `Authorization: Bearer <access-token>` header
// and attaches userID + claims to the gin context.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			unauthorized(c, "No token provided")
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenString == "" {
			unauthorized(c, "No token provided")
			return
		}

		claims, err := utils.ParseAccessToken(tokenString)
		if err != nil {
			unauthorized(c, "Invalid or expired token")
			return
		}

		userID, err := utils.GetUserIDFromClaims(claims)
		if err != nil {
			unauthorized(c, err.Error())
			return
		}

		c.Set("claims", claims)
		c.Set("userID", userID)
		c.Next()
	}
}

func unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, common.BaseApiResponse[any]{
		HttpRequestStatus: http.StatusUnauthorized,
		Success:           false,
		Message:           message,
		Data:              nil,
	})
	c.Abort()
}
