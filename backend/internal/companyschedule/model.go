// Package companyschedule manages the company-wide default weekly
// schedule: which shift governs each ISO weekday for every employee,
// unless overridden for an individual in internal/schedule
// (work_schedules).
//
// It sits between the per-employee override and the employee's own default
// shift in Phase 4's attendance resolution — see attendance.Service.
// resolveShift. A weekday configured with no shift (Day.ShiftID == nil) is
// a non-working day: a check-in then is refused, not measured against a
// fallback shift.
package companyschedule

import (
	"time"

	"github.com/google/uuid"
)

// Day is one weekday's entry in the company-wide default schedule.
// ShiftID nil means that weekday is a non-working day (libur).
type Day struct {
	DayOfWeek int
	ShiftID   *uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time

	// ShiftName is a denormalized display field populated by the repository's
	// LEFT JOIN on shifts; empty for a non-working day.
	ShiftName string
}
