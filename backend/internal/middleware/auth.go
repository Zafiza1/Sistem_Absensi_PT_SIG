package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/suryaintigas/absensi-backend/pkg/jwt"
	"github.com/suryaintigas/absensi-backend/pkg/response"
)

// Context keys populated by AuthRequired for downstream handlers and for
// RequireRole. Defined in the middleware package (rather than internal/auth)
// so that neither package needs to import the other: middleware depends on
// pkg/jwt and pkg/rbac only, and internal/auth depends on middleware for
// these keys.
const (
	ContextKeyUserID   = "auth_user_id"
	ContextKeyUserRole = "auth_user_role"
)

// AuthRequired validates the "Authorization: Bearer <token>" header using
// the given JWT manager. On success it stores the authenticated user's ID
// (string form of the UUID) and role in the request context. On failure it
// aborts the request with 401 before any handler runs.
func AuthRequired(manager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		token = strings.TrimSpace(token)
		if !ok || token == "" {
			response.Fail(c, http.StatusUnauthorized, "Token akses tidak ditemukan", nil)
			c.Abort()
			return
		}

		claims, err := manager.ParseAccessToken(token)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "Token akses tidak valid atau kedaluwarsa", nil)
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, claims.Subject)
		c.Set(ContextKeyUserRole, string(claims.Role))
		c.Next()
	}
}
