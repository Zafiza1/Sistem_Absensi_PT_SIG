package auth

import (
	"time"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/rbac"
)

// User is a dashboard/system account (Admin, HR, Management, ...).
// Distinct from an "employee" (Phase 3), who is tracked for attendance but
// never logs into the dashboard.
type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	Role         rbac.Role
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// RefreshToken is a server-tracked, revocable session token. Only its
// SHA-256 hash is ever persisted — see Service.hashToken.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	UserAgent string
	IPAddress string
	CreatedAt time.Time
}

// IsValid reports whether the token has not been revoked and has not
// expired as of now.
func (rt *RefreshToken) IsValid(now time.Time) bool {
	return rt.RevokedAt == nil && now.Before(rt.ExpiresAt)
}
