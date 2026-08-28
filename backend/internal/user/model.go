// Package user manages dashboard/system accounts (SUPER_ADMIN, ADMIN, HR,
// MANAGEMENT) for the admin dashboard's /users page. It operates on the
// same `users` table Phase 2's auth module authenticates against, but as
// its own module with a different concern: auth owns login/session
// mechanics (issuing and rotating tokens), user owns account lifecycle —
// who exists, their role, and whether they're still allowed to log in.
//
// The password hash never appears on this package's model: handlers and
// API responses built from User can't leak it even by a future mistake.
// Repository/Service methods that need to touch it (Create, ResetPassword)
// take or return it as a separate, explicit parameter instead.
package user

import (
	"time"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/rbac"
)

type User struct {
	ID        uuid.UUID
	Name      string
	Email     string
	Role      rbac.Role
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
