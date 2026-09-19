// Package payroll computes monthly salary deductions from attendance
// lateness, per company policy:
//
//	late  1-10 minutes  -> flat Rp 20,000
//	late 11-30 minutes  -> flat Rp 50,000
//	late 31+ minutes    -> half a day's pay (base_salary / 26 working days / 2)
//
// It has no rule for a check-in after 12:00 noon distinct from the 31+
// minutes tier — the source policy mentions noon only as the practical
// upper bound of when a late arrival is still a "late check-in" rather
// than a full-day absence, and internal/report already derives ABSENT for
// days with no attendance at all. A separate "checked in after noon"
// policy, if the company wants one, is not implemented here.
//
// It reads employees.base_salary ("Gaji Pokok", internal/employee) and
// attendances.late_minutes (internal/attendance), and cross-references
// internal/leave for how many of the employee's 12 annual leave days have
// been used, so a payroll run can be reviewed alongside remaining leave.
package payroll

import (
	"time"

	"github.com/google/uuid"
)

const (
	// LateTier1MaxMinutes is the upper bound (inclusive) of the first,
	// Rp 20,000 late-deduction tier.
	LateTier1MaxMinutes = 10
	LateTier1Amount     = 20000

	// LateTier2MaxMinutes is the upper bound (inclusive) of the second,
	// Rp 50,000 late-deduction tier. Above it, ComputeLateDeduction falls
	// through to the half-day tier.
	LateTier2MaxMinutes = 30
	LateTier2Amount     = 50000

	// WorkingDaysPerMonth is the divisor company policy uses to turn a
	// monthly base_salary into a daily rate for the half-day deduction.
	WorkingDaysPerMonth = 26
)

// ComputeLateDeduction returns the Rupiah deduction for one day's
// lateness. On-time (lateMinutes <= 0) returns 0.
func ComputeLateDeduction(lateMinutes int, baseSalary int64) int64 {
	switch {
	case lateMinutes <= 0:
		return 0
	case lateMinutes <= LateTier1MaxMinutes:
		return LateTier1Amount
	case lateMinutes <= LateTier2MaxMinutes:
		return LateTier2Amount
	default:
		return baseSalary / WorkingDaysPerMonth / 2
	}
}

// EmployeePayroll is one employee's deduction summary for the reported
// month.
type EmployeePayroll struct {
	EmployeeID     uuid.UUID
	EmployeeNumber string
	Name           string
	DepartmentName string
	BaseSalary     int64

	LateDays      int
	LateDeduction int64
	NetSalary     int64

	LeaveDaysUsed      int
	LeaveDaysRemaining int
}

// Monthly is the payroll report for one calendar month, optionally scoped
// to one department.
type Monthly struct {
	Year        int
	Month       int
	GeneratedAt time.Time
	Employees   []EmployeePayroll
}
