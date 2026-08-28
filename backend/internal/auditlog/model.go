// Package auditlog records who did what, when, across the system, backing
// the dashboard's /audit-logs page (Phase 6). Entries are append-only —
// there is deliberately no Update or Delete here, since an editable audit
// trail isn't one.
package auditlog

import (
	"time"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/rbac"
)

type Action string

const (
	ActionCreate      Action = "CREATE"
	ActionUpdate      Action = "UPDATE"
	ActionDelete      Action = "DELETE"
	ActionLogin       Action = "LOGIN"
	ActionLoginFailed Action = "LOGIN_FAILED"
)

// Entry is one immutable audit record. ActorName/ActorRole are denormalized
// snapshots taken at write time (not joined from `users` on read) so a log
// entry still reads correctly even if the actor's name/role later changes
// or their account is deleted — see the migration's comment on actor_id.
type Entry struct {
	ID          uuid.UUID
	ActorID     *uuid.UUID
	ActorName   string
	ActorRole   rbac.Role
	Action      Action
	EntityType  string
	EntityID    string
	Description string
	IPAddress   string
	CreatedAt   time.Time
}

// Filter narrows List by any combination of fields; zero values mean
// "don't filter on this".
type Filter struct {
	ActorID    *uuid.UUID
	EntityType string
	Action     Action
	DateFrom   *time.Time
	DateTo     *time.Time
}
