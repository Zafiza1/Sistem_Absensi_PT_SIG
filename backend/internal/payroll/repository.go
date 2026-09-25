package payroll

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EmployeeRow is the slice of an employee the monthly payroll report needs.
type EmployeeRow struct {
	ID             uuid.UUID
	EmployeeNumber string
	Name           string
	DepartmentName string
	BaseSalary     int64
}

// LateRow is one late attendance record in the reported month.
type LateRow struct {
	EmployeeID  uuid.UUID
	LateMinutes int
}

type Repository interface {
	// ActiveEmployees lists active (non-deleted) employees, ordered by
	// employee_number, optionally scoped to one department.
	ActiveEmployees(ctx context.Context, departmentID *uuid.UUID) ([]EmployeeRow, error)
	// LateAttendanceInRange returns every attendance row with
	// attendance_date in [from, to] and late_minutes > 0, optionally scoped
	// to one department.
	LateAttendanceInRange(ctx context.Context, from, to time.Time, departmentID *uuid.UUID) ([]LateRow, error)
	// LeaveDaysByEmployee sums leave.StatusActive days_count per employee
	// for leaves starting in year, optionally scoped to one department.
	LeaveDaysByEmployee(ctx context.Context, year int, departmentID *uuid.UUID) (map[uuid.UUID]int, error)

	// Payroll Period methods
	CreatePayrollPeriod(ctx context.Context, period *PayrollPeriod) error
	GetPayrollPeriod(ctx context.Context, id uuid.UUID) (*PayrollPeriod, error)
	GetPayrollPeriodByYearMonth(ctx context.Context, year, month int) (*PayrollPeriod, error)
	ListPayrollPeriods(ctx context.Context, year *int, status *string) ([]PayrollPeriod, error)
	UpdatePayrollPeriod(ctx context.Context, period *PayrollPeriod) error
	DeletePayrollPeriod(ctx context.Context, id uuid.UUID) error

	// Payroll Item methods
	CreatePayrollItem(ctx context.Context, item *PayrollItem) error
	GetPayrollItem(ctx context.Context, id uuid.UUID) (*PayrollItem, error)
	GetPayrollItemsByPeriod(ctx context.Context, periodID uuid.UUID) ([]PayrollItem, error)
	GetPayrollItemByEmployeePeriod(ctx context.Context, employeeID, periodID uuid.UUID) (*PayrollItem, error)
	UpdatePayrollItem(ctx context.Context, item *PayrollItem) error
	DeletePayrollItem(ctx context.Context, id uuid.UUID) error

	// Deduction Rule methods
	CreateDeductionRule(ctx context.Context, rule *DeductionRule) error
	GetDeductionRule(ctx context.Context, id uuid.UUID) (*DeductionRule, error)
	ListDeductionRules(ctx context.Context, ruleType *string, isActive *bool) ([]DeductionRule, error)
	UpdateDeductionRule(ctx context.Context, rule *DeductionRule) error
	DeleteDeductionRule(ctx context.Context, id uuid.UUID) error
	GetActiveDeductionRules(ctx context.Context) ([]DeductionRule, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) ActiveEmployees(ctx context.Context, departmentID *uuid.UUID) ([]EmployeeRow, error) {
	q := `
		SELECT e.id, e.employee_number, e.name, COALESCE(d.name, ''), e.base_salary
		FROM employees e
		LEFT JOIN departments d ON d.id = e.department_id
		WHERE e.deleted_at IS NULL AND e.status = 'ACTIVE'
		  AND ($1::uuid IS NULL OR e.department_id = $1)
		ORDER BY e.employee_number`

	rows, err := r.db.Query(ctx, q, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EmployeeRow
	for rows.Next() {
		var e EmployeeRow
		if err := rows.Scan(&e.ID, &e.EmployeeNumber, &e.Name, &e.DepartmentName, &e.BaseSalary); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) LateAttendanceInRange(ctx context.Context, from, to time.Time, departmentID *uuid.UUID) ([]LateRow, error) {
	q := `
		SELECT a.employee_id, a.late_minutes
		FROM attendances a
		WHERE a.attendance_date BETWEEN $1 AND $2 AND a.late_minutes > 0
		  AND ($3::uuid IS NULL OR a.employee_id IN (
		    SELECT id FROM employees WHERE department_id = $3
		  ))`

	rows, err := r.db.Query(ctx, q, from, to, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LateRow
	for rows.Next() {
		var l LateRow
		if err := rows.Scan(&l.EmployeeID, &l.LateMinutes); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) LeaveDaysByEmployee(ctx context.Context, year int, departmentID *uuid.UUID) (map[uuid.UUID]int, error) {
	q := `
		SELECT l.employee_id, SUM(l.days_count)
		FROM leaves l
		WHERE l.status = 'ACTIVE' AND EXTRACT(YEAR FROM l.start_date) = $1
		  AND ($2::uuid IS NULL OR l.employee_id IN (
		    SELECT id FROM employees WHERE department_id = $2
		  ))
		GROUP BY l.employee_id`

	rows, err := r.db.Query(ctx, q, year, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[uuid.UUID]int{}
	for rows.Next() {
		var (
			empID uuid.UUID
			days  int
		)
		if err := rows.Scan(&empID, &days); err != nil {
			return nil, err
		}
		out[empID] = days
	}
	return out, rows.Err()
}

// Payroll Period methods
func (r *PostgresRepository) CreatePayrollPeriod(ctx context.Context, period *PayrollPeriod) error {
	q := `
		INSERT INTO payroll_periods (id, period_start, period_end, year, month, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`

	err := r.db.QueryRow(ctx, q,
		period.ID, period.PeriodStart, period.PeriodEnd,
		period.Year, period.Month, period.Status, period.Notes,
	).Scan(&period.CreatedAt, &period.UpdatedAt)
	return err
}

func (r *PostgresRepository) GetPayrollPeriod(ctx context.Context, id uuid.UUID) (*PayrollPeriod, error) {
	q := `
		SELECT id, period_start, period_end, year, month, status, processed_at, processed_by,
		       total_employees, total_gross_pay, total_net_pay, total_deductions, notes, created_at, updated_at
		FROM payroll_periods WHERE id = $1`

	var period PayrollPeriod
	err := r.db.QueryRow(ctx, q, id).Scan(
		&period.ID, &period.PeriodStart, &period.PeriodEnd, &period.Year, &period.Month,
		&period.Status, &period.ProcessedAt, &period.ProcessedBy, &period.TotalEmployees,
		&period.TotalGrossPay, &period.TotalNetPay, &period.TotalDeductions,
		&period.Notes, &period.CreatedAt, &period.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &period, nil
}

func (r *PostgresRepository) GetPayrollPeriodByYearMonth(ctx context.Context, year, month int) (*PayrollPeriod, error) {
	q := `
		SELECT id, period_start, period_end, year, month, status, processed_at, processed_by,
		       total_employees, total_gross_pay, total_net_pay, total_deductions, notes, created_at, updated_at
		FROM payroll_periods WHERE year = $1 AND month = $2`

	var period PayrollPeriod
	err := r.db.QueryRow(ctx, q, year, month).Scan(
		&period.ID, &period.PeriodStart, &period.PeriodEnd, &period.Year, &period.Month,
		&period.Status, &period.ProcessedAt, &period.ProcessedBy, &period.TotalEmployees,
		&period.TotalGrossPay, &period.TotalNetPay, &period.TotalDeductions,
		&period.Notes, &period.CreatedAt, &period.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &period, nil
}

func (r *PostgresRepository) ListPayrollPeriods(ctx context.Context, year *int, status *string) ([]PayrollPeriod, error) {
	q := `
		SELECT id, period_start, period_end, year, month, status, processed_at, processed_by,
		       total_employees, total_gross_pay, total_net_pay, total_deductions, notes, created_at, updated_at
		FROM payroll_periods WHERE ($1::int IS NULL OR year = $1) AND ($2::varchar IS NULL OR status = $2)
		ORDER BY year DESC, month DESC`

	rows, err := r.db.Query(ctx, q, year, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var periods []PayrollPeriod
	for rows.Next() {
		var period PayrollPeriod
		if err := rows.Scan(
			&period.ID, &period.PeriodStart, &period.PeriodEnd, &period.Year, &period.Month,
			&period.Status, &period.ProcessedAt, &period.ProcessedBy, &period.TotalEmployees,
			&period.TotalGrossPay, &period.TotalNetPay, &period.TotalDeductions,
			&period.Notes, &period.CreatedAt, &period.UpdatedAt,
		); err != nil {
			return nil, err
		}
		periods = append(periods, period)
	}
	return periods, rows.Err()
}

func (r *PostgresRepository) UpdatePayrollPeriod(ctx context.Context, period *PayrollPeriod) error {
	q := `
		UPDATE payroll_periods 
		SET status = $2, processed_at = $3, processed_by = $4, total_employees = $5,
		    total_gross_pay = $6, total_net_pay = $7, total_deductions = $8, notes = $9, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`

	return r.db.QueryRow(ctx, q,
		period.ID, period.Status, period.ProcessedAt, period.ProcessedBy,
		period.TotalEmployees, period.TotalGrossPay, period.TotalNetPay,
		period.TotalDeductions, period.Notes,
	).Scan(&period.UpdatedAt)
}

func (r *PostgresRepository) DeletePayrollPeriod(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM payroll_periods WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}

// Payroll Item methods
func (r *PostgresRepository) CreatePayrollItem(ctx context.Context, item *PayrollItem) error {
	q := `
		INSERT INTO payroll_items (
			id, payroll_period_id, employee_id, employee_number, employee_name,
			department_id, department_name, position_id, position_name,
			working_days, present_days, absent_days, late_days, late_minutes, leave_days,
			base_salary, overtime_hours, overtime_pay, allowance, bonus, other_earnings, total_earnings,
			late_deduction, absent_deduction, tax_deduction, insurance_deduction, other_deductions, total_deductions,
			gross_pay, net_pay, payment_status, payment_date, payment_method, payment_reference, notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34
		) RETURNING created_at, updated_at`

	err := r.db.QueryRow(ctx, q,
		item.ID, item.PayrollPeriodID, item.EmployeeID, item.EmployeeNumber, item.EmployeeName,
		item.DepartmentID, item.DepartmentName, item.PositionID, item.PositionName,
		item.WorkingDays, item.PresentDays, item.AbsentDays, item.LateDays, item.LateMinutes, item.LeaveDays,
		item.BaseSalary, item.OvertimeHours, item.OvertimePay, item.Allowance, item.Bonus, item.OtherEarnings, item.TotalEarnings,
		item.LateDeduction, item.AbsentDeduction, item.TaxDeduction, item.InsuranceDeduction, item.OtherDeductions, item.TotalDeductions,
		item.GrossPay, item.NetPay, item.PaymentStatus, item.PaymentDate, item.PaymentMethod, item.PaymentReference, item.Notes,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
	return err
}

func (r *PostgresRepository) GetPayrollItem(ctx context.Context, id uuid.UUID) (*PayrollItem, error) {
	q := `
		SELECT id, payroll_period_id, employee_id, employee_number, employee_name,
		       department_id, department_name, position_id, position_name,
		       working_days, present_days, absent_days, late_days, late_minutes, leave_days,
		       base_salary, overtime_hours, overtime_pay, allowance, bonus, other_earnings, total_earnings,
		       late_deduction, absent_deduction, tax_deduction, insurance_deduction, other_deductions, total_deductions,
		       gross_pay, net_pay, payment_status, payment_date, payment_method, payment_reference, notes, created_at, updated_at
		FROM payroll_items WHERE id = $1`

	var item PayrollItem
	err := r.db.QueryRow(ctx, q, id).Scan(
		&item.ID, &item.PayrollPeriodID, &item.EmployeeID, &item.EmployeeNumber, &item.EmployeeName,
		&item.DepartmentID, &item.DepartmentName, &item.PositionID, &item.PositionName,
		&item.WorkingDays, &item.PresentDays, &item.AbsentDays, &item.LateDays, &item.LateMinutes, &item.LeaveDays,
		&item.BaseSalary, &item.OvertimeHours, &item.OvertimePay, &item.Allowance, &item.Bonus, &item.OtherEarnings, &item.TotalEarnings,
		&item.LateDeduction, &item.AbsentDeduction, &item.TaxDeduction, &item.InsuranceDeduction, &item.OtherDeductions, &item.TotalDeductions,
		&item.GrossPay, &item.NetPay, &item.PaymentStatus, &item.PaymentDate, &item.PaymentMethod, &item.PaymentReference, &item.Notes,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PostgresRepository) GetPayrollItemsByPeriod(ctx context.Context, periodID uuid.UUID) ([]PayrollItem, error) {
	q := `
		SELECT id, payroll_period_id, employee_id, employee_number, employee_name,
		       department_id, department_name, position_id, position_name,
		       working_days, present_days, absent_days, late_days, late_minutes, leave_days,
		       base_salary, overtime_hours, overtime_pay, allowance, bonus, other_earnings, total_earnings,
		       late_deduction, absent_deduction, tax_deduction, insurance_deduction, other_deductions, total_deductions,
		       gross_pay, net_pay, payment_status, payment_date, payment_method, payment_reference, notes, created_at, updated_at
		FROM payroll_items WHERE payroll_period_id = $1 ORDER BY employee_number`

	rows, err := r.db.Query(ctx, q, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PayrollItem
	for rows.Next() {
		var item PayrollItem
		if err := rows.Scan(
			&item.ID, &item.PayrollPeriodID, &item.EmployeeID, &item.EmployeeNumber, &item.EmployeeName,
			&item.DepartmentID, &item.DepartmentName, &item.PositionID, &item.PositionName,
			&item.WorkingDays, &item.PresentDays, &item.AbsentDays, &item.LateDays, &item.LateMinutes, &item.LeaveDays,
			&item.BaseSalary, &item.OvertimeHours, &item.OvertimePay, &item.Allowance, &item.Bonus, &item.OtherEarnings, &item.TotalEarnings,
			&item.LateDeduction, &item.AbsentDeduction, &item.TaxDeduction, &item.InsuranceDeduction, &item.OtherDeductions, &item.TotalDeductions,
			&item.GrossPay, &item.NetPay, &item.PaymentStatus, &item.PaymentDate, &item.PaymentMethod, &item.PaymentReference, &item.Notes,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) GetPayrollItemByEmployeePeriod(ctx context.Context, employeeID, periodID uuid.UUID) (*PayrollItem, error) {
	q := `
		SELECT id, payroll_period_id, employee_id, employee_number, employee_name,
		       department_id, department_name, position_id, position_name,
		       working_days, present_days, absent_days, late_days, late_minutes, leave_days,
		       base_salary, overtime_hours, overtime_pay, allowance, bonus, other_earnings, total_earnings,
		       late_deduction, absent_deduction, tax_deduction, insurance_deduction, other_deductions, total_deductions,
		       gross_pay, net_pay, payment_status, payment_date, payment_method, payment_reference, notes, created_at, updated_at
		FROM payroll_items WHERE employee_id = $1 AND payroll_period_id = $2`

	var item PayrollItem
	err := r.db.QueryRow(ctx, q, employeeID, periodID).Scan(
		&item.ID, &item.PayrollPeriodID, &item.EmployeeID, &item.EmployeeNumber, &item.EmployeeName,
		&item.DepartmentID, &item.DepartmentName, &item.PositionID, &item.PositionName,
		&item.WorkingDays, &item.PresentDays, &item.AbsentDays, &item.LateDays, &item.LateMinutes, &item.LeaveDays,
		&item.BaseSalary, &item.OvertimeHours, &item.OvertimePay, &item.Allowance, &item.Bonus, &item.OtherEarnings, &item.TotalEarnings,
		&item.LateDeduction, &item.AbsentDeduction, &item.TaxDeduction, &item.InsuranceDeduction, &item.OtherDeductions, &item.TotalDeductions,
		&item.GrossPay, &item.NetPay, &item.PaymentStatus, &item.PaymentDate, &item.PaymentMethod, &item.PaymentReference, &item.Notes,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PostgresRepository) UpdatePayrollItem(ctx context.Context, item *PayrollItem) error {
	q := `
		UPDATE payroll_items 
		SET working_days = $2, present_days = $3, absent_days = $4, late_days = $5, late_minutes = $6, leave_days = $7,
		    base_salary = $8, overtime_hours = $9, overtime_pay = $10, allowance = $11, bonus = $12, other_earnings = $13, total_earnings = $14,
		    late_deduction = $15, absent_deduction = $16, tax_deduction = $17, insurance_deduction = $18, other_deductions = $19, total_deductions = $20,
		    gross_pay = $21, net_pay = $22, payment_status = $23, payment_date = $24, payment_method = $25, payment_reference = $26, notes = $27, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`

	return r.db.QueryRow(ctx, q,
		item.ID, item.WorkingDays, item.PresentDays, item.AbsentDays, item.LateDays, item.LateMinutes, item.LeaveDays,
		item.BaseSalary, item.OvertimeHours, item.OvertimePay, item.Allowance, item.Bonus, item.OtherEarnings, item.TotalEarnings,
		item.LateDeduction, item.AbsentDeduction, item.TaxDeduction, item.InsuranceDeduction, item.OtherDeductions, item.TotalDeductions,
		item.GrossPay, item.NetPay, item.PaymentStatus, item.PaymentDate, item.PaymentMethod, item.PaymentReference, item.Notes,
	).Scan(&item.UpdatedAt)
}

func (r *PostgresRepository) DeletePayrollItem(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM payroll_items WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}

// Deduction Rule methods
func (r *PostgresRepository) CreateDeductionRule(ctx context.Context, rule *DeductionRule) error {
	q := `
		INSERT INTO payroll_deduction_rules (
			id, rule_type, rule_name, description, late_min_minutes, late_max_minutes,
			deduction_amount, deduction_type, percentage_value, is_active, effective_date, expiry_date, priority
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at`

	err := r.db.QueryRow(ctx, q,
		rule.ID, rule.RuleType, rule.RuleName, rule.Description, rule.LateMinMinutes, rule.LateMaxMinutes,
		rule.DeductionAmount, rule.DeductionType, rule.PercentageValue, rule.IsActive, rule.EffectiveDate, rule.ExpiryDate, rule.Priority,
	).Scan(&rule.CreatedAt, &rule.UpdatedAt)
	return err
}

func (r *PostgresRepository) GetDeductionRule(ctx context.Context, id uuid.UUID) (*DeductionRule, error) {
	q := `
		SELECT id, rule_type, rule_name, description, late_min_minutes, late_max_minutes,
		       deduction_amount, deduction_type, percentage_value, is_active, effective_date, expiry_date, priority, created_at, updated_at
		FROM payroll_deduction_rules WHERE id = $1`

	var rule DeductionRule
	err := r.db.QueryRow(ctx, q, id).Scan(
		&rule.ID, &rule.RuleType, &rule.RuleName, &rule.Description, &rule.LateMinMinutes, &rule.LateMaxMinutes,
		&rule.DeductionAmount, &rule.DeductionType, &rule.PercentageValue, &rule.IsActive, &rule.EffectiveDate, &rule.ExpiryDate, &rule.Priority,
		&rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *PostgresRepository) ListDeductionRules(ctx context.Context, ruleType *string, isActive *bool) ([]DeductionRule, error) {
	q := `
		SELECT id, rule_type, rule_name, description, late_min_minutes, late_max_minutes,
		       deduction_amount, deduction_type, percentage_value, is_active, effective_date, expiry_date, priority, created_at, updated_at
		FROM payroll_deduction_rules 
		WHERE ($1::varchar IS NULL OR rule_type = $1) AND ($2::boolean IS NULL OR is_active = $2)
		ORDER BY priority DESC, created_at DESC`

	rows, err := r.db.Query(ctx, q, ruleType, isActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []DeductionRule
	for rows.Next() {
		var rule DeductionRule
		if err := rows.Scan(
			&rule.ID, &rule.RuleType, &rule.RuleName, &rule.Description, &rule.LateMinMinutes, &rule.LateMaxMinutes,
			&rule.DeductionAmount, &rule.DeductionType, &rule.PercentageValue, &rule.IsActive, &rule.EffectiveDate, &rule.ExpiryDate, &rule.Priority,
			&rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *PostgresRepository) UpdateDeductionRule(ctx context.Context, rule *DeductionRule) error {
	q := `
		UPDATE payroll_deduction_rules 
		SET rule_type = $2, rule_name = $3, description = $4, late_min_minutes = $5, late_max_minutes = $6,
		    deduction_amount = $7, deduction_type = $8, percentage_value = $9, is_active = $10, 
		    effective_date = $11, expiry_date = $12, priority = $13, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`

	return r.db.QueryRow(ctx, q,
		rule.ID, rule.RuleType, rule.RuleName, rule.Description, rule.LateMinMinutes, rule.LateMaxMinutes,
		rule.DeductionAmount, rule.DeductionType, rule.PercentageValue, rule.IsActive, rule.EffectiveDate, rule.ExpiryDate, rule.Priority,
	).Scan(&rule.UpdatedAt)
}

func (r *PostgresRepository) DeleteDeductionRule(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM payroll_deduction_rules WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}

func (r *PostgresRepository) GetActiveDeductionRules(ctx context.Context) ([]DeductionRule, error) {
	q := `
		SELECT id, rule_type, rule_name, description, late_min_minutes, late_max_minutes,
		       deduction_amount, deduction_type, percentage_value, is_active, effective_date, expiry_date, priority, created_at, updated_at
		FROM payroll_deduction_rules 
		WHERE is_active = true 
		  AND (expiry_date IS NULL OR expiry_date > CURRENT_DATE)
		  AND effective_date <= CURRENT_DATE
		ORDER BY priority DESC`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []DeductionRule
	for rows.Next() {
		var rule DeductionRule
		if err := rows.Scan(
			&rule.ID, &rule.RuleType, &rule.RuleName, &rule.Description, &rule.LateMinMinutes, &rule.LateMaxMinutes,
			&rule.DeductionAmount, &rule.DeductionType, &rule.PercentageValue, &rule.IsActive, &rule.EffectiveDate, &rule.ExpiryDate, &rule.Priority,
			&rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

var _ Repository = (*PostgresRepository)(nil)
