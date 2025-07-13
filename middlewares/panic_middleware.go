package middlewares

import (
	"fmt"
	"j_ai_trade/common"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Middleware to recover from panics
func PanicRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				var panicMessage string
				if err, ok := r.(error); ok {
					panicMessage = err.Error()
				} else {
					panicMessage = fmt.Sprint(r)
				}

				c.JSON(http.StatusBadRequest, common.BaseApiResponse[any]{
					HttpRequestStatus: http.StatusBadRequest,
					Success:           false,
					Message:           panicMessage,
					Data:              nil,
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}
