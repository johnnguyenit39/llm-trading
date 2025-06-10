package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"time"
)

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Retrieve the Request-ID from the context
		requestID := c.GetString(RequestIDKey)

		// Process the request
		c.Next()

		// Log the request details with Request-ID
		log.Info().
			Str("method", method).
			Str("path", path).
			Str("request_id", requestID).
			Int("status", c.Writer.Status()).
			Dur("duration", time.Since(start)).
			Msg("Handled request")
	}
}
