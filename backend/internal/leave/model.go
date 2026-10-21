// Package leave manages excused absences: cuti tahunan (annual leave), izin
// and sakit. Rows are recorded by HR/Admin on the dashboard — there is no
// employee self-service flow, the same posture as internal/employee and
// internal/device. HR learns of an izin/sakit from a letter or verbally, so
// the dashboard's daily report lets them label a no-check-in day in a click.
// Company policy caps cuti (only) at AnnualQuotaDays per employee per
// calendar year; see Service.Request for how that's enforced.
package leave

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive    = "ACTIVE"
	StatusCancelled = "CANCELLED"
)

// Leave types. Only TypeCuti counts against AnnualQuotaDays; izin and sakit
// are unlimited but, unlike cuti, are deducted from salary in payroll.
const (
	TypeCuti  = "CUTI"
	TypeIzin  = "IZIN"
	TypeSakit = "SAKIT"
)

// ValidType reports whether t is one of the leave types above.
func ValidType(t string) bool {
	return t == TypeCuti || t == TypeIzin || t == TypeSakit
}

// AnnualQuotaDays is the number of paid annual leave days an employee is
// entitled to per calendar year, per company policy.
const AnnualQuotaDays = 12

// Leave is one recorded leave period for an employee.
type Leave struct {
	ID         uuid.UUID
	EmployeeID uuid.UUID
	Type       string
	StartDate  time.Time
	EndDate    time.Time
	DaysCount  int
	Reason     string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// Denormalized display fields populated by JOINs in the repository.
	EmployeeName   string
	EmployeeNumber string
}
