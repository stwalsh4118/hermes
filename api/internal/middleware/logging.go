// Package middleware provides HTTP middleware functions for request logging and processing.
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stwalsh4118/hermes/internal/logger"
)

// RequestLogger returns a Gin middleware for logging HTTP requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Capture request start time
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Log request with structured fields
		logger.Log.Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", c.Writer.Status()).
			Dur("duration", duration).
			Str("client_ip", c.ClientIP()).
			Msg("HTTP request")

		// Log errors separately if any occurred during request processing
		if len(c.Errors) > 0 {
			logger.Log.Error().
				Strs("errors", c.Errors.Errors()).
				Str("path", path).
				Msg("Request completed with errors")
		}
	}
}
