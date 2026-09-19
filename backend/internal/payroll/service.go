package payroll

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/leave"
)

var ErrInvalidMonth = errors.New("payroll: month must be between 1 and 12")

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
// (1-12), optionally scoped to one department.
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
