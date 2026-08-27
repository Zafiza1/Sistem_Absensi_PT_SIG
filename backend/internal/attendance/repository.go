package attendance

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suryaintigas/absensi-backend/pkg/dberr"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

var (
	ErrNotFound          = errors.New("attendance: not found")
	ErrDuplicateCheckIn  = errors.New("attendance: employee already has an attendance record for this date")
	ErrAlreadyCheckedOut = errors.New("attendance: already checked out")
)

// Filter narrows List() results. Zero-value fields are ignored.
type Filter struct {
	EmployeeID *uuid.UUID
	Status     string
	DateFrom   *time.Time
	DateTo     *time.Time
}

type Repository interface {
	Create(ctx context.Context, a *Attendance) error
	FindByID(ctx context.Context, id uuid.UUID) (*Attendance, error)
	// FindOpenByEmployee returns the employee's most recent attendance row
	// with no check-out yet, regardless of attendance_date — an overnight
	// shift's check-out can land on the calendar day after check-in, so
	// check-out deliberately does not require an exact date match.
	FindOpenByEmployee(ctx context.Context, employeeID uuid.UUID) (*Attendance, error)
	CompleteCheckOut(ctx context.Context, id uuid.UUID, checkOutAt time.Time, deviceID uuid.UUID, workingDurationMinutes int) error
	List(ctx context.Context, f Filter, p pagination.Params) ([]Attendance, int64, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// attendanceColumns and attendanceFrom are kept separate (rather than one
// combined constant) so List() can splice ", COUNT(*) OVER() AS total"
// into the SELECT column list — appending it after a combined
// "SELECT ... FROM ... JOIN ..." string would land it after the JOINs
// instead, producing invalid SQL (this bit Phase 3's employee/schedule
// repositories; see their comments).
const attendanceColumns = `
	a.id, a.employee_id, a.shift_id, a.attendance_date,
	a.check_in_at, a.check_in_device_id, a.check_out_at, a.check_out_device_id,
	a.status, a.late_minutes, a.working_duration_minutes,
	a.created_at, a.updated_at,
	e.name, e.employee_number, COALESCE(sh.name, ''),
	COALESCE(cid.device_name, ''), COALESCE(cod.device_name, '')`

const attendanceFrom = `
	FROM attendances a
	JOIN employees e ON e.id = a.employee_id
	LEFT JOIN shifts sh ON sh.id = a.shift_id
	LEFT JOIN devices cid ON cid.id = a.check_in_device_id
	LEFT JOIN devices cod ON cod.id = a.check_out_device_id`

const baseSelect = "SELECT " + attendanceColumns + attendanceFrom

func scanAttendance(row pgx.Row, a *Attendance) error {
	return row.Scan(
		&a.ID, &a.EmployeeID, &a.ShiftID, &a.AttendanceDate,
		&a.CheckInAt, &a.CheckInDeviceID, &a.CheckOutAt, &a.CheckOutDeviceID,
		&a.Status, &a.LateMinutes, &a.WorkingDurationMinutes,
		&a.CreatedAt, &a.UpdatedAt,
		&a.EmployeeName, &a.EmployeeNumber, &a.ShiftName,
		&a.CheckInDeviceName, &a.CheckOutDeviceName,
	)
}

func (r *PostgresRepository) Create(ctx context.Context, a *Attendance) error {
	const q = `
		INSERT INTO attendances (employee_id, shift_id, attendance_date, check_in_at, check_in_device_id, status, late_minutes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, q, a.EmployeeID, a.ShiftID, a.AttendanceDate, a.CheckInAt, a.CheckInDeviceID, a.Status, a.LateMinutes).
		Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if dberr.IsUniqueViolation(err) {
		return ErrDuplicateCheckIn
	}
	if err != nil {
		return err
	}
	return r.reload(ctx, a)
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Attendance, error) {
	q := baseSelect + ` WHERE a.id = $1`

	var a Attendance
	err := scanAttendance(r.db.QueryRow(ctx, q, id), &a)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *PostgresRepository) FindOpenByEmployee(ctx context.Context, employeeID uuid.UUID) (*Attendance, error) {
	q := baseSelect + ` WHERE a.employee_id = $1 AND a.check_out_at IS NULL ORDER BY a.check_in_at DESC LIMIT 1`

	var a Attendance
	err := scanAttendance(r.db.QueryRow(ctx, q, employeeID), &a)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *PostgresRepository) CompleteCheckOut(ctx context.Context, id uuid.UUID, checkOutAt time.Time, deviceID uuid.UUID, workingDurationMinutes int) error {
	const q = `
		UPDATE attendances
		SET check_out_at = $1, check_out_device_id = $2, working_duration_minutes = $3, status = 'CHECKED_OUT'
		WHERE id = $4 AND check_out_at IS NULL`

	tag, err := r.db.Exec(ctx, q, checkOutAt, deviceID, workingDurationMinutes, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAlreadyCheckedOut
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, f Filter, p pagination.Params) ([]Attendance, int64, error) {
	var (
		conditions []string
		args       []any
	)
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	if f.EmployeeID != nil {
		conditions = append(conditions, "a.employee_id = "+arg(*f.EmployeeID))
	}
	if f.Status != "" {
		conditions = append(conditions, "a.status = "+arg(f.Status))
	}
	if f.DateFrom != nil {
		conditions = append(conditions, "a.attendance_date >= "+arg(*f.DateFrom))
	}
	if f.DateTo != nil {
		conditions = append(conditions, "a.attendance_date <= "+arg(*f.DateTo))
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	q := "SELECT " + attendanceColumns + ", COUNT(*) OVER() AS total" + attendanceFrom + where +
		" ORDER BY a.attendance_date DESC, a.check_in_at DESC LIMIT " + arg(p.PageSize) + " OFFSET " + arg(p.Offset())

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		items []Attendance
		total int64
	)
	for rows.Next() {
		var a Attendance
		if err := rows.Scan(
			&a.ID, &a.EmployeeID, &a.ShiftID, &a.AttendanceDate,
			&a.CheckInAt, &a.CheckInDeviceID, &a.CheckOutAt, &a.CheckOutDeviceID,
			&a.Status, &a.LateMinutes, &a.WorkingDurationMinutes,
			&a.CreatedAt, &a.UpdatedAt,
			&a.EmployeeName, &a.EmployeeNumber, &a.ShiftName,
			&a.CheckInDeviceName, &a.CheckOutDeviceName, &total,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// reload re-fetches a by ID and overwrites it in place, populating the
// employee/shift/device display names joined in from other tables — the
// plain INSERT ... RETURNING above only has the raw attendances row
// available, not those joins.
func (r *PostgresRepository) reload(ctx context.Context, a *Attendance) error {
	full, err := r.FindByID(ctx, a.ID)
	if err != nil {
		return err
	}
	*a = *full
	return nil
}

var _ Repository = (*PostgresRepository)(nil)
