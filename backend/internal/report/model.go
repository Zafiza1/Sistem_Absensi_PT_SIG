// Package report builds the dashboard's aggregated attendance reports.
//
// The monthly report is the one place ABSENT is materialised: the backend
// never writes an ABSENT attendance row (see internal/attendance's package
// doc), so "did not come in" is derived here by walking every active
// employee across every calendar day of the month and checking whether that
// day was a working day for them — resolved the same three-layer way
// attendance.resolveShift does it (per-employee schedule override →
// company-wide schedule → the employee's own default shift). A working day
// with no check-in that HR has labelled cuti, izin or sakit (internal/leave)
// counts as that instead, so ABSENT means alpha: no word from the employee.
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
	DayAbsent  DayStatus = "ABSENT"  // a past working day with no check-in and no leave (alpha)
	DayCuti    DayStatus = "CUTI"    // a working day covered by an active cuti
	DayIzin    DayStatus = "IZIN"    // a working day covered by an active izin
	DaySakit   DayStatus = "SAKIT"   // a working day covered by an active sakit
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

	Totals

	Days []DayCell `json:"days"`
}

// Totals are the per-status day counts HR reads, shared by the monthly and
// yearly reports.
type Totals struct {
	WorkingDays int `json:"working_days"` // OnTime + LateCount + Izin + Sakit + Cuti + Absent
	OnTime      int `json:"on_time"`
	LateCount   int `json:"late_count"`
	LateMinutes int `json:"late_minutes"`
	Izin        int `json:"izin"`
	Sakit       int `json:"sakit"`
	Cuti        int `json:"cuti"`
	Absent      int `json:"absent"` // alpha
}

func (t *Totals) add(o Totals) {
	t.WorkingDays += o.WorkingDays
	t.OnTime += o.OnTime
	t.LateCount += o.LateCount
	t.LateMinutes += o.LateMinutes
	t.Izin += o.Izin
	t.Sakit += o.Sakit
	t.Cuti += o.Cuti
	t.Absent += o.Absent
}

// Monthly is the whole report for one month.
type Monthly struct {
	Year        int              `json:"year"`
	Month       int              `json:"month"` // 1-12
	DaysInMonth int              `json:"days_in_month"`
	GeneratedAt time.Time        `json:"generated_at"`
	Employees   []EmployeeReport `json:"employees"`
}

// Headcount is "how many people, how many days" for one absence category —
// the figure HR used to tally by hand across twelve monthly spreadsheets.
type Headcount struct {
	People int `json:"people"`
	Days   int `json:"days"`
}

// CategoryCounts groups a Headcount per category HR recaps.
type CategoryCounts struct {
	Late   Headcount `json:"late"`
	Izin   Headcount `json:"izin"`
	Sakit  Headcount `json:"sakit"`
	Cuti   Headcount `json:"cuti"`
	Absent Headcount `json:"absent"`
}

// YearlyEmployee is one employee's year: the year's totals plus a Totals
// per month (index 0 = January).
type YearlyEmployee struct {
	EmployeeID     uuid.UUID `json:"employee_id"`
	EmployeeNumber string    `json:"employee_number"`
	Name           string    `json:"name"`
	DepartmentName string    `json:"department_name"`

	Totals

	Months []Totals `json:"months"`
}

// YearMonth is one month's company-wide headcounts in the yearly report.
type YearMonth struct {
	Month int `json:"month"`
	CategoryCounts
}

// Yearly is the year-end recap: per-category headcounts for the year and
// per month, per-employee totals, and the twelve underlying monthly reports
// (kept for the Excel export's per-month sheets, not sent as JSON).
type Yearly struct {
	Year        int              `json:"year"`
	GeneratedAt time.Time        `json:"generated_at"`
	Summary     CategoryCounts   `json:"summary"`
	Months      []YearMonth      `json:"months"`
	Employees   []YearlyEmployee `json:"employees"`

	Monthly []*Monthly `json:"-"`
}
