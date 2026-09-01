package companyschedule

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suryaintigas/absensi-backend/pkg/dberr"
)

var (
	// ErrNotFound means that weekday has no company_schedules row — the
	// caller (attendance.resolveShift) should fall through to the employee's
	// default shift.
	ErrNotFound = errors.New("company schedule: weekday not configured")
	// ErrInvalidShift is returned by Replace when a supplied shift_id does
	// not exist.
	ErrInvalidShift = errors.New("company schedule: shift not found")
)

type Repository interface {
	// List returns every configured weekday, ordered by day_of_week.
	List(ctx context.Context) ([]Day, error)
	// FindByDay returns one weekday's entry, or ErrNotFound if that weekday
	// is not configured. A configured non-working day is returned with a nil
	// ShiftID (not ErrNotFound).
	FindByDay(ctx context.Context, dayOfWeek int) (*Day, error)
	// Replace atomically swaps the whole table for days: every existing row
	// is removed and the supplied entries inserted. A day absent from days
	// becomes "not configured" again.
	Replace(ctx context.Context, days []Day) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const selectColumns = `
	cs.day_of_week, cs.shift_id, cs.created_at, cs.updated_at, COALESCE(s.name, '')
	FROM company_schedules cs
	LEFT JOIN shifts s ON s.id = cs.shift_id`

func scanDay(row pgx.Row, d *Day) error {
	return row.Scan(&d.DayOfWeek, &d.ShiftID, &d.CreatedAt, &d.UpdatedAt, &d.ShiftName)
}

func (r *PostgresRepository) List(ctx context.Context) ([]Day, error) {
	rows, err := r.db.Query(ctx, `SELECT `+selectColumns+` ORDER BY cs.day_of_week`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var days []Day
	for rows.Next() {
		var d Day
		if err := scanDay(rows, &d); err != nil {
			return nil, err
		}
		days = append(days, d)
	}
	return days, rows.Err()
}

func (r *PostgresRepository) FindByDay(ctx context.Context, dayOfWeek int) (*Day, error) {
	var d Day
	err := scanDay(r.db.QueryRow(ctx, `SELECT `+selectColumns+` WHERE cs.day_of_week = $1`, dayOfWeek), &d)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *PostgresRepository) Replace(ctx context.Context, days []Day) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a successful Commit

	if _, err := tx.Exec(ctx, `DELETE FROM company_schedules`); err != nil {
		return err
	}

	for _, d := range days {
		_, err := tx.Exec(ctx,
			`INSERT INTO company_schedules (day_of_week, shift_id) VALUES ($1, $2)`,
			d.DayOfWeek, d.ShiftID)
		if dberr.IsForeignKeyViolation(err) {
			return ErrInvalidShift
		}
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

var _ Repository = (*PostgresRepository)(nil)
