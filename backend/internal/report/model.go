// Package report builds the dashboard's aggregated attendance reports.
//
// The monthly report is the one place ABSENT is materialised: the backend
// never writes an ABSENT attendance row (see internal/attendance's package
// doc), so "did not come in" is derived here by walking every active
// employee across every calendar day of the month and checking whether that
// day was a working day for them — resolved the same three-layer way
// attendance.resolveShift does it (per-employee schedule override →
// company-wide schedule → the employee's own default shift).
package report

import (
	"time"

	"github.com/google/uuid"
)

// DayStatus is the outcome of one employee on one calendar day.
type DayStatus string

const (
	DayOnTime  DayStatus = "ON_TIME" // checked in, not late
	DayLate    DayStatus = "LATE"    // checked in past shift start + tolerance
	DayAbsent  DayStatus = "ABSENT"  // a past working day with no check-in
	DayOff     DayStatus = "OFF"     // not a working day (weekend / libur / no shift)
	DayPending DayStatus = "PENDING" // a working day that is today or still ahead
)

// DayCell is one employee-day in the detail grid.
type DayCell struct {
	Day         int        `json:"day"` // day of month, 1-based
	Status      DayStatus  `json:"status"`
	LateMinutes int        `json:"late_minutes"`
	CheckInAt   *time.Time `json:"check_in_at"`
	CheckOutAt  *time.Time `json:"check_out_at"`
}

// EmployeeReport is one row of the monthly report: the per-day grid plus
// the month totals HR actually reads.
type EmployeeReport struct {
	EmployeeID     uuid.UUID `json:"employee_id"`
	EmployeeNumber string    `json:"employee_number"`
	Name           string    `json:"name"`
	DepartmentName string    `json:"department_name"`

	WorkingDays int `json:"working_days"` // elapsed working days = OnTime + LateCount + Absent
	OnTime      int `json:"on_time"`
	LateCount   int `json:"late_count"`
	LateMinutes int `json:"late_minutes"` // total across the month
	Absent      int `json:"absent"`

	Days []DayCell `json:"days"`
}

// Monthly is the whole report for one month.
type Monthly struct {
	Year        int              `json:"year"`
	Month       int              `json:"month"` // 1-12
	DaysInMonth int              `json:"days_in_month"`
	GeneratedAt time.Time        `json:"generated_at"`
	Employees   []EmployeeReport `json:"employees"`
}
