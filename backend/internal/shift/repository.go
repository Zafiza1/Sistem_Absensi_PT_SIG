package shift

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
	ErrNotFound      = errors.New("shift: not found")
	ErrNameTaken     = errors.New("shift: name already in use")
	ErrHasReferences = errors.New("shift: still referenced by other records")
)

type Repository interface {
	Create(ctx context.Context, s *Shift) error
	FindByID(ctx context.Context, id uuid.UUID) (*Shift, error)
	List(ctx context.Context, p pagination.Params) ([]Shift, int64, error)
	Update(ctx context.Context, s *Shift) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const selectColumns = `id, name, start_time, end_time, is_overnight, late_tolerance_minutes, working_duration_minutes, is_active, created_at, updated_at`

func scanShift(row pgx.Row, s *Shift) error {
	return row.Scan(
		&s.ID, &s.Name, &s.StartTime, &s.EndTime, &s.IsOvernight,
		&s.LateToleranceMinutes, &s.WorkingDurationMinutes, &s.IsActive,
		&s.CreatedAt, &s.UpdatedAt,
	)
}

func (r *PostgresRepository) Create(ctx context.Context, s *Shift) error {
	const q = `
		INSERT INTO shifts (name, start_time, end_time, is_overnight, late_tolerance_minutes, working_duration_minutes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, q, s.Name, s.StartTime, s.EndTime, s.IsOvernight, s.LateToleranceMinutes, s.WorkingDurationMinutes, s.IsActive).
		Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if dberr.IsUniqueViolation(err) {
		return ErrNameTaken
	}
	return err
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Shift, error) {
	q := `SELECT ` + selectColumns + ` FROM shifts WHERE id = $1`

	var s Shift
	err := scanShift(r.db.QueryRow(ctx, q, id), &s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepository) List(ctx context.Context, p pagination.Params) ([]Shift, int64, error) {
	q := `SELECT ` + selectColumns + `, COUNT(*) OVER() AS total FROM shifts ORDER BY name LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, q, p.PageSize, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		items []Shift
		total int64
	)
	for rows.Next() {
		var s Shift
		if err := rows.Scan(
			&s.ID, &s.Name, &s.StartTime, &s.EndTime, &s.IsOvernight,
			&s.LateToleranceMinutes, &s.WorkingDurationMinutes, &s.IsActive,
			&s.CreatedAt, &s.UpdatedAt, &total,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, s *Shift) error {
	const q = `
		UPDATE shifts
		SET name = $1, start_time = $2, end_time = $3, is_overnight = $4,
		    late_tolerance_minutes = $5, working_duration_minutes = $6, is_active = $7
		WHERE id = $8
		RETURNING updated_at`

	err := r.db.QueryRow(ctx, q, s.Name, s.StartTime, s.EndTime, s.IsOvernight,
		s.LateToleranceMinutes, s.WorkingDurationMinutes, s.IsActive, s.ID).
		Scan(&s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if dberr.IsUniqueViolation(err) {
		return ErrNameTaken
	}
	return err
}

// Delete refuses when active employees default to this shift, or when any
// work_schedules row still references it (work_schedules.shift_id is ON
// DELETE RESTRICT, but employees.shift_id is SET NULL — so only the
// work_schedules side would naturally fail at the FK; the employees check
// is added explicitly for the same reason as department/position).
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `
		DELETE FROM shifts
		WHERE id = $1
		  AND NOT EXISTS (
		    SELECT 1 FROM employees WHERE employees.shift_id = $1 AND employees.deleted_at IS NULL
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM work_schedules WHERE work_schedules.shift_id = $1
		  )`

	tag, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	if _, findErr := r.FindByID(ctx, id); findErr != nil {
		return findErr
	}
	return ErrHasReferences
}

var _ Repository = (*PostgresRepository)(nil)
