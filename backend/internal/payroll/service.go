package payroll

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/leave"
)

var (
	ErrInvalidMonth  = errors.New("payroll: month must be between 1 and 12")
	ErrPeriodExists  = errors.New("payroll: period for this year/month already exists")
	ErrPeriodLocked  = errors.New("payroll: period is locked and cannot be modified")
	ErrInvalidStatus = errors.New("payroll: invalid status transition")
)

// jakarta matches internal/attendance and internal/report: every
// date-range boundary here is evaluated in the company's single operating
// timezone, regardless of the server's own.
var jakarta = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}()

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Monthly builds the payroll deduction report for the given year and month
// (1-12), optionally scoped to one department (legacy compatibility).
func (s *Service) Monthly(ctx context.Context, year, month int, departmentID *uuid.UUID) (*Monthly, error) {
	if month < 1 || month > 12 {
		return nil, ErrInvalidMonth
	}

	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, jakarta)
	daysInMonth := first.AddDate(0, 1, -1).Day()
	last := time.Date(year, time.Month(month), daysInMonth, 0, 0, 0, 0, jakarta)

	employees, err := s.repo.ActiveEmployees(ctx, departmentID)
	if err != nil {
		return nil, err
	}
	lateRows, err := s.repo.LateAttendanceInRange(ctx, first, last, departmentID)
	if err != nil {
		return nil, err
	}
	leaveDays, err := s.repo.LeaveDaysByEmployee(ctx, year, departmentID)
	if err != nil {
		return nil, err
	}

	report := &Monthly{
		Year:        year,
		Month:       month,
		GeneratedAt: time.Now().In(jakarta),
		Employees:   make([]EmployeePayroll, 0, len(employees)),
	}

	for _, emp := range employees {
		ep := EmployeePayroll{
			EmployeeID:     emp.ID,
			EmployeeNumber: emp.EmployeeNumber,
			Name:           emp.Name,
			DepartmentName: emp.DepartmentName,
			BaseSalary:     emp.BaseSalary,
		}

		for _, l := range lateRows {
			if l.EmployeeID != emp.ID {
				continue
			}
			ep.LateDays++
			ep.LateDeduction += ComputeLateDeduction(l.LateMinutes, emp.BaseSalary)
		}
		ep.NetSalary = emp.BaseSalary - ep.LateDeduction

		ep.LeaveDaysUsed = leaveDays[emp.ID]
		ep.LeaveDaysRemaining = leave.AnnualQuotaDays - ep.LeaveDaysUsed
		if ep.LeaveDaysRemaining < 0 {
			ep.LeaveDaysRemaining = 0
		}

		report.Employees = append(report.Employees, ep)
	}

	return report, nil
}

// Payroll Period Management

// CreatePayrollPeriod creates a new payroll period for the specified year and month
func (s *Service) CreatePayrollPeriod(ctx context.Context, year, month int, notes string) (*PayrollPeriod, error) {
	if month < 1 || month > 12 {
		return nil, ErrInvalidMonth
	}

	// Check if period already exists
	existing, err := s.repo.GetPayrollPeriodByYearMonth(ctx, year, month)
	if err == nil && existing != nil {
		return nil, ErrPeriodExists
	}

	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, jakarta)
	daysInMonth := first.AddDate(0, 1, -1).Day()
	last := time.Date(year, time.Month(month), daysInMonth, 23, 59, 59, 0, jakarta)

	period := &PayrollPeriod{
		ID:          uuid.New(),
		PeriodStart: first,
		PeriodEnd:   last,
		Year:        year,
		Month:       month,
		Status:      "DRAFT",
		Notes:       notes,
	}

	if err := s.repo.CreatePayrollPeriod(ctx, period); err != nil {
		return nil, fmt.Errorf("failed to create payroll period: %w", err)
	}

	return period, nil
}

// GetPayrollPeriod retrieves a payroll period by ID
func (s *Service) GetPayrollPeriod(ctx context.Context, id uuid.UUID) (*PayrollPeriod, error) {
	return s.repo.GetPayrollPeriod(ctx, id)
}

// ListPayrollPeriods lists payroll periods with optional filters
func (s *Service) ListPayrollPeriods(ctx context.Context, year *int, status *string) ([]PayrollPeriod, error) {
	return s.repo.ListPayrollPeriods(ctx, year, status)
}

// ProcessPayrollPeriod processes payroll for a period
func (s *Service) ProcessPayrollPeriod(ctx context.Context, periodID uuid.UUID, processorID uuid.UUID) (*PayrollPeriod, error) {
	period, err := s.repo.GetPayrollPeriod(ctx, periodID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payroll period: %w", err)
	}

	if period.Status == "LOCKED" {
		return nil, ErrPeriodLocked
	}
	if period.Status != "DRAFT" {
		return nil, ErrInvalidStatus
	}

	// Update status to PROCESSING
	period.Status = "PROCESSING"
	if err := s.repo.UpdatePayrollPeriod(ctx, period); err != nil {
		return nil, fmt.Errorf("failed to update period status: %w", err)
	}

	// Generate payroll items for all active employees
	employees, err := s.repo.ActiveEmployees(ctx, nil)
	if err != nil {
		period.Status = "DRAFT"
		s.repo.UpdatePayrollPeriod(ctx, period)
		return nil, fmt.Errorf("failed to get employees: %w", err)
	}

	// Get attendance data for the period
	lateRows, err := s.repo.LateAttendanceInRange(ctx, period.PeriodStart, period.PeriodEnd, nil)
	if err != nil {
		period.Status = "DRAFT"
		s.repo.UpdatePayrollPeriod(ctx, period)
		return nil, fmt.Errorf("failed to get attendance data: %w", err)
	}

	var totalGrossPay, totalNetPay, totalDeductions int64
	var processedCount int

	for _, emp := range employees {
		item := &PayrollItem{
			ID:              uuid.New(),
			PayrollPeriodID: periodID,
			EmployeeID:      emp.ID,
			EmployeeNumber:  emp.EmployeeNumber,
			EmployeeName:    emp.Name,
			DepartmentName:  emp.DepartmentName,
			BaseSalary:      emp.BaseSalary,
		}

		// Calculate attendance summary
		var totalLateMinutes int
		var lateDays int

		for _, l := range lateRows {
			if l.EmployeeID == emp.ID {
				lateDays++
				totalLateMinutes += l.LateMinutes
			}
		}

		item.LateDays = lateDays
		item.LateMinutes = totalLateMinutes
		item.PresentDays = lateDays // Simplified - in production, calculate properly

		// Calculate deductions
		item.LateDeduction = s.calculateLateDeduction(ctx, totalLateMinutes, emp.BaseSalary)
		item.TotalDeductions = item.LateDeduction

		// Calculate earnings
		item.TotalEarnings = emp.BaseSalary
		item.GrossPay = emp.BaseSalary
		item.NetPay = item.GrossPay - item.TotalDeductions

		// Save payroll item
		if err := s.repo.CreatePayrollItem(ctx, item); err != nil {
			continue // Skip this employee but continue with others
		}

		totalGrossPay += item.GrossPay
		totalNetPay += item.NetPay
		totalDeductions += item.TotalDeductions
		processedCount++
	}

	// Update period totals
	period.TotalEmployees = processedCount
	period.TotalGrossPay = totalGrossPay
	period.TotalNetPay = totalNetPay
	period.TotalDeductions = totalDeductions
	period.Status = "COMPLETED"
	now := time.Now().In(jakarta)
	period.ProcessedAt = &now
	period.ProcessedBy = &processorID

	if err := s.repo.UpdatePayrollPeriod(ctx, period); err != nil {
		return nil, fmt.Errorf("failed to update completed period: %w", err)
	}

	return period, nil
}

// LockPayrollPeriod locks a payroll period to prevent further modifications
func (s *Service) LockPayrollPeriod(ctx context.Context, periodID uuid.UUID) error {
	period, err := s.repo.GetPayrollPeriod(ctx, periodID)
	if err != nil {
		return fmt.Errorf("failed to get payroll period: %w", err)
	}

	if period.Status != "COMPLETED" {
		return ErrInvalidStatus
	}

	period.Status = "LOCKED"
	return s.repo.UpdatePayrollPeriod(ctx, period)
}

// DeletePayrollPeriod deletes a payroll period (only allowed if DRAFT)
func (s *Service) DeletePayrollPeriod(ctx context.Context, periodID uuid.UUID) error {
	period, err := s.repo.GetPayrollPeriod(ctx, periodID)
	if err != nil {
		return fmt.Errorf("failed to get payroll period: %w", err)
	}

	if period.Status != "DRAFT" {
		return ErrInvalidStatus
	}

	return s.repo.DeletePayrollPeriod(ctx, periodID)
}

// Payroll Item Management

// GetPayrollItem retrieves a single payroll item by ID
func (s *Service) GetPayrollItem(ctx context.Context, id uuid.UUID) (*PayrollItem, error) {
	return s.repo.GetPayrollItem(ctx, id)
}

// GetPayrollItemsByPeriod retrieves all payroll items for a period
func (s *Service) GetPayrollItemsByPeriod(ctx context.Context, periodID uuid.UUID) ([]PayrollItem, error) {
	return s.repo.GetPayrollItemsByPeriod(ctx, periodID)
}

// UpdatePayrollItem updates a payroll item
func (s *Service) UpdatePayrollItem(ctx context.Context, item *PayrollItem) error {
	// Recalculate totals
	item.TotalEarnings = item.BaseSalary + item.OvertimePay + item.Allowance + item.Bonus + item.OtherEarnings
	item.TotalDeductions = item.LateDeduction + item.AbsentDeduction + item.TaxDeduction + item.InsuranceDeduction + item.OtherDeductions
	item.GrossPay = item.TotalEarnings
	item.NetPay = item.GrossPay - item.TotalDeductions

	return s.repo.UpdatePayrollItem(ctx, item)
}

// MarkAsPaid marks payroll items as paid
func (s *Service) MarkAsPaid(ctx context.Context, periodID uuid.UUID, paymentDate time.Time, paymentMethod, reference string) error {
	items, err := s.repo.GetPayrollItemsByPeriod(ctx, periodID)
	if err != nil {
		return fmt.Errorf("failed to get payroll items: %w", err)
	}

	for _, item := range items {
		item.PaymentStatus = "PAID"
		item.PaymentDate = &paymentDate
		item.PaymentMethod = paymentMethod
		item.PaymentReference = reference
		if err := s.repo.UpdatePayrollItem(ctx, &item); err != nil {
			return fmt.Errorf("failed to update payroll item: %w", err)
		}
	}

	return nil
}

// Deduction Rule Management

// CreateDeductionRule creates a new deduction rule
func (s *Service) CreateDeductionRule(ctx context.Context, rule *DeductionRule) error {
	return s.repo.CreateDeductionRule(ctx, rule)
}

// GetDeductionRule retrieves a single deduction rule by ID
func (s *Service) GetDeductionRule(ctx context.Context, id uuid.UUID) (*DeductionRule, error) {
	return s.repo.GetDeductionRule(ctx, id)
}

// ListDeductionRules lists deduction rules with optional filters
func (s *Service) ListDeductionRules(ctx context.Context, ruleType *string, isActive *bool) ([]DeductionRule, error) {
	return s.repo.ListDeductionRules(ctx, ruleType, isActive)
}

// UpdateDeductionRule updates a deduction rule
func (s *Service) UpdateDeductionRule(ctx context.Context, rule *DeductionRule) error {
	return s.repo.UpdateDeductionRule(ctx, rule)
}

// DeleteDeductionRule deletes a deduction rule
func (s *Service) DeleteDeductionRule(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteDeductionRule(ctx, id)
}

// Helper methods

func (s *Service) calculateLateDeduction(ctx context.Context, totalLateMinutes int, baseSalary int64) int64 {
	var totalDeduction int64

	// Get active deduction rules
	rules, err := s.repo.GetActiveDeductionRules(ctx)
	if err != nil {
		// Fallback to default calculation
		return ComputeLateDeduction(totalLateMinutes, baseSalary)
	}

	// Calculate using rules (simplified - in production, implement proper rule matching)
	for _, lateMinutes := range s.splitLateMinutes(totalLateMinutes) {
		for _, rule := range rules {
			if rule.RuleType == "LATE" {
				if s.matchesLateRule(&rule, lateMinutes) {
					deduction := s.calculateRuleDeduction(&rule, baseSalary)
					totalDeduction += deduction
					break // Use first matching rule
				}
			}
		}
	}

	return totalDeduction
}

func (s *Service) splitLateMinutes(totalLateMinutes int) []int {
	// Simplified - split into individual late instances
	// In production, this should be based on actual attendance records
	if totalLateMinutes == 0 {
		return []int{}
	}
	return []int{totalLateMinutes}
}

func (s *Service) matchesLateRule(rule *DeductionRule, lateMinutes int) bool {
	if rule.LateMinMinutes == nil && rule.LateMaxMinutes == nil {
		return true
	}

	min := 0
	if rule.LateMinMinutes != nil {
		min = *rule.LateMinMinutes
	}

	max := 999999
	if rule.LateMaxMinutes != nil {
		max = *rule.LateMaxMinutes
	}

	return lateMinutes >= min && lateMinutes <= max
}

func (s *Service) calculateRuleDeduction(rule *DeductionRule, baseSalary int64) int64 {
	switch rule.DeductionType {
	case "FIXED":
		return rule.DeductionAmount
	case "PERCENTAGE":
		if rule.PercentageValue != nil {
			return int64(float64(baseSalary) * (*rule.PercentageValue) / 100)
		}
		return rule.DeductionAmount
	case "HALF_DAY_SALARY":
		return baseSalary / WorkingDaysPerMonth / 2
	default:
		return rule.DeductionAmount
	}
}
