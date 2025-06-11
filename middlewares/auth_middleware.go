package middlewares

import (
	"j-ai-trade/common"
	"j-ai-trade/utils"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "No token provided",
				Data:              nil,
			})
			c.Abort()
			return
		}

		tokenString := authHeader
		token, err := utils.ParseJWT(tokenString)
		if err != nil || !token.Valid {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           "Invalid token",
				Data:              nil,
			})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("claims", claims)
		}

		userId, err := utils.GetUserIDromJWT(authHeader)
		if err != nil {
			c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusBadRequest,
				Success:           false,
				Message:           err.Error(),
				Data:              nil,
			})
			c.Abort()
			return
		}
		c.Set("userID", userId)
		c.Next()
	}
}
