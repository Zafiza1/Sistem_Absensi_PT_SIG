// Package shift manages work shifts (Pagi/Siang/Malam or any admin-defined
// shift), each with a start/end time, late tolerance, and expected working
// duration used by Phase 4's attendance calculations.
package shift

import (
	"time"

	"github.com/google/uuid"
)

// Shift's StartTime/EndTime are stored and exchanged as "HH:MM" strings
// (see Service.parseClock) rather than time.Time, since a shift is a daily
// recurring time-of-day, not a specific date.
type Shift struct {
	ID                     uuid.UUID
	Name                   string
	StartTime              string // "HH:MM"
	EndTime                string // "HH:MM"
	IsOvernight            bool
	LateToleranceMinutes   int
	WorkingDurationMinutes int
	IsActive               bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
