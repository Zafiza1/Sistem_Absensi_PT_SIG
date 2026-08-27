package schedule

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suryaintigas/absensi-backend/pkg/dberr"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

var (
	ErrNotFound         = errors.New("schedule: not found")
	ErrAlreadyAssigned  = errors.New("schedule: employee already has a shift assigned for this day")
	ErrInvalidReference = errors.New("schedule: employee or shift not found")
)

type Repository interface {
	Create(ctx context.Context, s *Schedule) error
	FindByID(ctx context.Context, id uuid.UUID) (*Schedule, error)
	ListByEmployee(ctx context.Context, employeeID uuid.UUID) ([]Schedule, error)
	List(ctx context.Context, p pagination.Params) ([]Schedule, int64, error)
	Update(ctx context.Context, s *Schedule) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// scheduleColumns and scheduleFrom are kept separate (rather than one
// combined constant) so List() can splice ", COUNT(*) OVER() AS total"
// into the SELECT column list — appending it after a combined
// "SELECT ... FROM ... JOIN ..." string would land it after the JOINs
// instead, producing invalid SQL.
const scheduleColumns = `
	ws.id, ws.employee_id, ws.shift_id, ws.day_of_week, ws.created_at, ws.updated_at,
	e.name, s.name`

const scheduleFrom = `
	FROM work_schedules ws
	JOIN employees e ON e.id = ws.employee_id
	JOIN shifts s ON s.id = ws.shift_id`

const baseSelect = "SELECT " + scheduleColumns + scheduleFrom

func scanSchedule(row pgx.Row, sc *Schedule) error {
	return row.Scan(&sc.ID, &sc.EmployeeID, &sc.ShiftID, &sc.DayOfWeek, &sc.CreatedAt, &sc.UpdatedAt,
		&sc.EmployeeName, &sc.ShiftName)
}

func (r *PostgresRepository) Create(ctx context.Context, s *Schedule) error {
	const q = `
		INSERT INTO work_schedules (employee_id, shift_id, day_of_week)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, q, s.EmployeeID, s.ShiftID, s.DayOfWeek).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if dberr.IsUniqueViolation(err) {
		return ErrAlreadyAssigned
	}
	if dberr.IsForeignKeyViolation(err) {
		return ErrInvalidReference
	}
	if err != nil {
		return err
	}
	return r.reload(ctx, s)
}

// reload re-fetches s by ID and overwrites it in place, populating the
// employee/shift display names joined in from other tables — the plain
// INSERT/UPDATE ... RETURNING above only has the raw work_schedules row
// available, not those joins.
func (r *PostgresRepository) reload(ctx context.Context, s *Schedule) error {
	full, err := r.FindByID(ctx, s.ID)
	if err != nil {
		return err
	}
	*s = *full
	return nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Schedule, error) {
	q := baseSelect + ` WHERE ws.id = $1`

	var s Schedule
	err := scanSchedule(r.db.QueryRow(ctx, q, id), &s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepository) ListByEmployee(ctx context.Context, employeeID uuid.UUID) ([]Schedule, error) {
	q := baseSelect + ` WHERE ws.employee_id = $1 ORDER BY ws.day_of_week`

	rows, err := r.db.Query(ctx, q, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Schedule
	for rows.Next() {
		var s Schedule
		if err := scanSchedule(rows, &s); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) List(ctx context.Context, p pagination.Params) ([]Schedule, int64, error) {
	q := "SELECT " + scheduleColumns + ", COUNT(*) OVER() AS total" + scheduleFrom + `
		ORDER BY e.name, ws.day_of_week
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, q, p.PageSize, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		items []Schedule
		total int64
	)
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(&s.ID, &s.EmployeeID, &s.ShiftID, &s.DayOfWeek, &s.CreatedAt, &s.UpdatedAt,
			&s.EmployeeName, &s.ShiftName, &total); err != nil {
			return nil, 0, err
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, s *Schedule) error {
	const q = `
		UPDATE work_schedules
		SET shift_id = $1, day_of_week = $2
		WHERE id = $3
		RETURNING updated_at`

	err := r.db.QueryRow(ctx, q, s.ShiftID, s.DayOfWeek, s.ID).Scan(&s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if dberr.IsUniqueViolation(err) {
		return ErrAlreadyAssigned
	}
	if dberr.IsForeignKeyViolation(err) {
		return ErrInvalidReference
	}
	if err != nil {
		return err
	}
	return r.reload(ctx, s)
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM work_schedules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

var _ Repository = (*PostgresRepository)(nil)
