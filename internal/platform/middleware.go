package platform

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// info del request
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		c.Next() // procesar el request

		statusCode := c.Writer.Status()
		latency := time.Since(start)                               // cuanto tardo
		errorMsg := c.Errors.ByType(gin.ErrorTypePrivate).String() // error si hubo

		// determinar nivel  de log segun status code
		logEvent := log.Info() // determinar nivel de log segun status code
		if statusCode >= 500 {
			logEvent = log.Error()
		} else if statusCode >= 400 {
			logEvent = log.Warn()
		}

		logEvent.
			Str("method", method).
			Str("path", path).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("ip", clientIP).
			Str("user_agent", c.Request.UserAgent())

		if errorMsg != "" {
			logEvent.Str("error", errorMsg)
		}

		logEvent.Msg("HTTP Request")
	}
}
