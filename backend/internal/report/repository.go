package report

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EmployeeRow is the slice of an employee the monthly report needs.
type EmployeeRow struct {
	ID             uuid.UUID
	EmployeeNumber string
	Name           string
	DepartmentName string
	ShiftID        *uuid.UUID // the employee's own default shift, or nil
}

// AttendanceRow is one attendance record in the reported month.
type AttendanceRow struct {
	EmployeeID  uuid.UUID
	Date        time.Time // attendance_date (a DATE — use Day()/Month()/Year() only)
	CheckInAt   *time.Time
	CheckOutAt  *time.Time
	LateMinutes int
}

// LeaveRow is one active cuti/izin/sakit overlapping the reported range.
type LeaveRow struct {
	EmployeeID uuid.UUID
	Type       string    // CUTI, IZIN or SAKIT
	StartDate  time.Time // DATE — use Day()/Month()/Year() only
	EndDate    time.Time // DATE, inclusive
}

type Repository interface {
	// ActiveEmployees lists active (non-deleted) employees, ordered by
	// employee_number, optionally scoped to one department.
	ActiveEmployees(ctx context.Context, departmentID *uuid.UUID) ([]EmployeeRow, error)
	// ScheduleOverrideDays returns, per employee, the ISO weekdays that
	// employee has a work_schedules row for. A work_schedules row always
	// carries a shift, so its mere presence means "working day".
	ScheduleOverrideDays(ctx context.Context) (map[uuid.UUID]map[int]bool, error)
	// CompanyScheduleDays maps an ISO weekday to whether the company-wide
	// schedule makes it a working day (true) or an explicit day off (false).
	// A weekday absent from the map is "not configured".
	CompanyScheduleDays(ctx context.Context) (map[int]bool, error)
	// AttendanceInRange returns every attendance row with attendance_date in
	// [from, to], optionally scoped to one department.
	AttendanceInRange(ctx context.Context, from, to time.Time, departmentID *uuid.UUID) ([]AttendanceRow, error)
	// LeavesInRange returns every ACTIVE leave whose [start_date, end_date]
	// overlaps [from, to], optionally scoped to one department.
	LeavesInRange(ctx context.Context, from, to time.Time, departmentID *uuid.UUID) ([]LeaveRow, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) ActiveEmployees(ctx context.Context, departmentID *uuid.UUID) ([]EmployeeRow, error) {
	q := `
		SELECT e.id, e.employee_number, e.name, COALESCE(d.name, ''), e.shift_id
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
		if err := rows.Scan(&e.ID, &e.EmployeeNumber, &e.Name, &e.DepartmentName, &e.ShiftID); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ScheduleOverrideDays(ctx context.Context) (map[uuid.UUID]map[int]bool, error) {
	rows, err := r.db.Query(ctx, `SELECT employee_id, day_of_week FROM work_schedules`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[uuid.UUID]map[int]bool{}
	for rows.Next() {
		var empID uuid.UUID
		var day int
		if err := rows.Scan(&empID, &day); err != nil {
			return nil, err
		}
		if out[empID] == nil {
			out[empID] = map[int]bool{}
		}
		out[empID][day] = true
	}
	return out, rows.Err()
}

func (r *PostgresRepository) CompanyScheduleDays(ctx context.Context) (map[int]bool, error) {
	rows, err := r.db.Query(ctx, `SELECT day_of_week, shift_id IS NOT NULL FROM company_schedules`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int]bool{}
	for rows.Next() {
		var day int
		var working bool
		if err := rows.Scan(&day, &working); err != nil {
			return nil, err
		}
		out[day] = working
	}
	return out, rows.Err()
}

func (r *PostgresRepository) AttendanceInRange(ctx context.Context, from, to time.Time, departmentID *uuid.UUID) ([]AttendanceRow, error) {
	q := `
		SELECT a.employee_id, a.attendance_date, a.check_in_at, a.check_out_at, a.late_minutes
		FROM attendances a
		WHERE a.attendance_date BETWEEN $1 AND $2
		  AND ($3::uuid IS NULL OR a.employee_id IN (
		    SELECT id FROM employees WHERE department_id = $3
		  ))`

	rows, err := r.db.Query(ctx, q, from, to, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AttendanceRow
	for rows.Next() {
		var a AttendanceRow
		if err := rows.Scan(&a.EmployeeID, &a.Date, &a.CheckInAt, &a.CheckOutAt, &a.LateMinutes); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) LeavesInRange(ctx context.Context, from, to time.Time, departmentID *uuid.UUID) ([]LeaveRow, error) {
	q := `
		SELECT l.employee_id, l.leave_type, l.start_date, l.end_date
		FROM leaves l
		WHERE l.status = 'ACTIVE' AND l.start_date <= $2 AND l.end_date >= $1
		  AND ($3::uuid IS NULL OR l.employee_id IN (
		    SELECT id FROM employees WHERE department_id = $3
		  ))`

	rows, err := r.db.Query(ctx, q, from, to, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LeaveRow
	for rows.Next() {
		var l LeaveRow
		if err := rows.Scan(&l.EmployeeID, &l.Type, &l.StartDate, &l.EndDate); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

var _ Repository = (*PostgresRepository)(nil)
