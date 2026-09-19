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

var _ Repository = (*PostgresRepository)(nil)
