package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDHeader is the HTTP header carrying the correlation ID for a
// request, both inbound (client-supplied, e.g. from the Flutter app or a
// reverse proxy) and outbound (echoed back so clients can log it too).
const RequestIDHeader = "X-Request-ID"

// ContextKeyRequestID is the gin.Context key the request ID is stored
// under, for retrieval in handlers/services that need to thread it into
// logs or audit entries.
const ContextKeyRequestID = "request_id"

// RequestID assigns a correlation ID to every request: it reuses one
// supplied by the client/proxy, or generates a new UUID otherwise. The ID is
// echoed in the response header and made available via
// c.GetString(middleware.ContextKeyRequestID).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(ContextKeyRequestID, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}
