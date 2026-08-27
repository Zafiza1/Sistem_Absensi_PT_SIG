package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/suryaintigas/absensi-backend/pkg/rbac"
	"github.com/suryaintigas/absensi-backend/pkg/response"
)

// RequireRole restricts a route to the given roles. It must be registered
// after AuthRequired(), which populates ContextKeyUserRole.
func RequireRole(roles ...rbac.Role) gin.HandlerFunc {
	allowed := make(map[rbac.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		role := rbac.Role(c.GetString(ContextKeyUserRole))
		if _, ok := allowed[role]; !ok {
			response.Fail(c, http.StatusForbidden, "Anda tidak memiliki akses untuk aksi ini", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
