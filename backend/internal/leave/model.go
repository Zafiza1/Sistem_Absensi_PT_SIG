// Package leave manages cuti tahunan (annual leave). Rows are recorded by
// HR/Admin on the dashboard — there is no employee self-service flow, the
// same posture as internal/employee and internal/device. Company policy
// caps annual leave at AnnualQuotaDays per employee per calendar year; see
// Service.Request for how that's enforced.
package leave

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive    = "ACTIVE"
	StatusCancelled = "CANCELLED"
)

// AnnualQuotaDays is the number of paid annual leave days an employee is
// entitled to per calendar year, per company policy.
const AnnualQuotaDays = 12

// Leave is one recorded leave period for an employee.
type Leave struct {
	ID         uuid.UUID
	EmployeeID uuid.UUID
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
