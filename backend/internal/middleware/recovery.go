package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/suryaintigas/absensi-backend/pkg/response"
)

// Recovery catches panics in downstream handlers, logs them with a stack
// trace, and returns the standard JSON error envelope instead of letting
// Gin's default recovery write a bare 500 with no consistent body shape.
//
// The panic value and stack trace are only ever written to the server log,
// never to the HTTP response — a raw panic message can contain internal
// implementation details (file paths, SQL, struct contents) that must not
// leak to a client.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic_recovered",
					slog.String("request_id", c.GetString(ContextKeyRequestID)),
					slog.Any("error", rec),
					slog.String("stack", string(debug.Stack())),
				)
				response.Fail(c, http.StatusInternalServerError,
					"Terjadi kesalahan pada server. Silakan coba lagi.", nil)
				c.Abort()
			}
		}()
		c.Next()
	}
}
