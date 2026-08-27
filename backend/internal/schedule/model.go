// Package schedule manages per-employee, per-weekday shift overrides
// (work_schedules). An employee with no schedule row for a given day falls
// back to their default shift (employee.Employee.ShiftID) in Phase 4's
// attendance logic.
package schedule

import (
	"time"

	"github.com/google/uuid"
)

// DayOfWeek follows ISO 8601: 1 = Monday ... 7 = Sunday.
type Schedule struct {
	ID         uuid.UUID
	EmployeeID uuid.UUID
	ShiftID    uuid.UUID
	DayOfWeek  int
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// Denormalized display fields populated by JOINs in List/FindByID.
	EmployeeName string
	ShiftName    string
}
