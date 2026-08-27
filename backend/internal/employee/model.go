// Package employee manages the people whose attendance is tracked —
// distinct from internal/auth's User, which is a dashboard login account.
package employee

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"
)

// Employee is the core master-data record every attendance check-in/out
// (Phase 4) and face profile (Phase 5) attaches to.
type Employee struct {
	ID             uuid.UUID
	EmployeeNumber string
	Name           string
	Email          *string
	Phone          *string
	DepartmentID   *uuid.UUID
	PositionID     *uuid.UUID
	ShiftID        *uuid.UUID
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time

	// Denormalized display names, populated by List/FindByID via a JOIN so
	// the dashboard doesn't need three extra round-trips per row. Empty
	// when the corresponding *ID is nil.
	DepartmentName string
	PositionName   string
	ShiftName      string
}
