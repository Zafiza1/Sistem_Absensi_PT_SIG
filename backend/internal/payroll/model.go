// Package payroll provides comprehensive payroll management including
// attendance-based deductions, salary calculations, and payroll period management.
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

// PayrollPeriod represents a payroll processing period (usually monthly)
type PayrollPeriod struct {
	ID              uuid.UUID
	PeriodStart     time.Time
	PeriodEnd       time.Time
	Year            int
	Month           int
	Status          string // DRAFT, PROCESSING, COMPLETED, LOCKED
	ProcessedAt     *time.Time
	ProcessedBy     *uuid.UUID
	TotalEmployees  int
	TotalGrossPay   int64
	TotalNetPay     int64
	TotalDeductions int64
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PayrollItem represents payroll data for a single employee in a period
type PayrollItem struct {
	ID              uuid.UUID
	PayrollPeriodID uuid.UUID
	EmployeeID      uuid.UUID
	EmployeeNumber  string
	EmployeeName    string
	DepartmentID    *uuid.UUID
	DepartmentName  string
	PositionID      *uuid.UUID
	PositionName    string

	// Attendance summary
	WorkingDays int
	PresentDays int
	AbsentDays  int
	LateDays    int
	LateMinutes int
	LeaveDays   int

	// Earnings
	BaseSalary    int64
	OvertimeHours float64
	OvertimePay   int64
	Allowance     int64
	Bonus         int64
	OtherEarnings int64
	TotalEarnings int64

	// Deductions
	LateDeduction      int64
	AbsentDeduction    int64
	TaxDeduction       int64
	InsuranceDeduction int64
	OtherDeductions    int64
	TotalDeductions    int64

	// Final amounts
	GrossPay int64
	NetPay   int64

	// Payment info
	PaymentStatus    string // PENDING, PAID, FAILED
	PaymentDate      *time.Time
	PaymentMethod    string
	PaymentReference string
	Notes            string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// DeductionRule represents a payroll deduction rule
type DeductionRule struct {
	ID              uuid.UUID
	RuleType        string // LATE, ABSENT, OTHER
	RuleName        string
	Description     string
	LateMinMinutes  *int
	LateMaxMinutes  *int
	DeductionAmount int64
	DeductionType   string // FIXED, PERCENTAGE, HALF_DAY_SALARY
	PercentageValue *float64
	IsActive        bool
	EffectiveDate   time.Time
	ExpiryDate      *time.Time
	Priority        int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// EmployeePayroll is one employee's deduction summary for the reported
// month (legacy compatibility)
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
// to one department (legacy compatibility)
type Monthly struct {
	Year        int
	Month       int
	GeneratedAt time.Time
	Employees   []EmployeePayroll
}
