package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger emits one structured log line per HTTP request, tagged with
// the correlation ID set by RequestID(). Put RequestID() before this
// middleware in the chain.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		attrs := []any{
			slog.String("request_id", c.GetString(ContextKeyRequestID)),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("client_ip", c.ClientIP()),
		}

		switch {
		case status >= 500:
			logger.Error("http_request", attrs...)
		case status >= 400:
			logger.Warn("http_request", attrs...)
		default:
			logger.Info("http_request", attrs...)
		}
	}
}
