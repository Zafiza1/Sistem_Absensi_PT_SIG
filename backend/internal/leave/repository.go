package leave

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
	ErrNotFound         = errors.New("leave: not found")
	ErrInvalidEmployee  = errors.New("leave: employee not found")
	ErrAlreadyCancelled = errors.New("leave: already cancelled")
	ErrOverlap          = errors.New("leave: overlaps an existing active leave")
)

// Filter narrows List() results. Zero-value fields are ignored.
type Filter struct {
	EmployeeID *uuid.UUID
	Status     string
	Type       string
	// DateFrom/DateTo keep leaves whose [start_date, end_date] overlaps the
	// given range (either bound may be nil).
	DateFrom *time.Time
	DateTo   *time.Time
}

type Repository interface {
	Create(ctx context.Context, l *Leave) error
	FindByID(ctx context.Context, id uuid.UUID) (*Leave, error)
	List(ctx context.Context, f Filter, p pagination.Params) ([]Leave, int64, error)
	Cancel(ctx context.Context, id uuid.UUID) error
	// SumActiveDays returns the total days_count of ACTIVE cuti rows for
	// employeeID whose start_date falls in calendar year — the figure
	// Service.Request checks against AnnualQuotaDays. Izin/sakit rows are
	// excluded: they don't consume the quota.
	SumActiveDays(ctx context.Context, employeeID uuid.UUID, year int) (int, error)
	// HasActiveOverlap reports whether employeeID already has an ACTIVE
	// leave (of any type) covering any day in [start, end].
	HasActiveOverlap(ctx context.Context, employeeID uuid.UUID, start, end time.Time) (bool, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const leaveColumns = `
	l.id, l.employee_id, l.leave_type, l.start_date, l.end_date, l.days_count, l.reason, l.status,
	l.created_at, l.updated_at, e.name, e.employee_number`

const leaveFrom = `
	FROM leaves l
	JOIN employees e ON e.id = l.employee_id`

const baseSelect = "SELECT " + leaveColumns + leaveFrom

func scanLeave(row pgx.Row, l *Leave) error {
	var reason *string
	err := row.Scan(
		&l.ID, &l.EmployeeID, &l.Type, &l.StartDate, &l.EndDate, &l.DaysCount, &reason, &l.Status,
		&l.CreatedAt, &l.UpdatedAt, &l.EmployeeName, &l.EmployeeNumber,
	)
	if reason != nil {
		l.Reason = *reason
	}
	return err
}

func (r *PostgresRepository) Create(ctx context.Context, l *Leave) error {
	const q = `
		INSERT INTO leaves (employee_id, leave_type, start_date, end_date, days_count, reason, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, q, l.EmployeeID, l.Type, l.StartDate, l.EndDate, l.DaysCount, nullIfEmpty(l.Reason), l.Status).
		Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
	if dberr.IsForeignKeyViolation(err) {
		return ErrInvalidEmployee
	}
	if err != nil {
		return err
	}
	return r.reload(ctx, l)
}

func (r *PostgresRepository) reload(ctx context.Context, l *Leave) error {
	full, err := r.FindByID(ctx, l.ID)
	if err != nil {
		return err
	}
	*l = *full
	return nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Leave, error) {
	q := baseSelect + ` WHERE l.id = $1`

	var l Leave
	err := scanLeave(r.db.QueryRow(ctx, q, id), &l)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *PostgresRepository) List(ctx context.Context, f Filter, p pagination.Params) ([]Leave, int64, error) {
	var (
		conditions = []string{"1=1"}
		args       []any
	)
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	if f.EmployeeID != nil {
		conditions = append(conditions, "l.employee_id = "+arg(*f.EmployeeID))
	}
	if f.Status != "" {
		conditions = append(conditions, "l.status = "+arg(f.Status))
	}
	if f.Type != "" {
		conditions = append(conditions, "l.leave_type = "+arg(f.Type))
	}
	if f.DateFrom != nil {
		conditions = append(conditions, "l.end_date >= "+arg(*f.DateFrom))
	}
	if f.DateTo != nil {
		conditions = append(conditions, "l.start_date <= "+arg(*f.DateTo))
	}

	q := "SELECT " + leaveColumns + ", COUNT(*) OVER() AS total" + leaveFrom +
		" WHERE " + strings.Join(conditions, " AND ") +
		" ORDER BY l.start_date DESC LIMIT " + arg(p.PageSize) + " OFFSET " + arg(p.Offset())

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		items []Leave
		total int64
	)
	for rows.Next() {
		var (
			l      Leave
			reason *string
		)
		if err := rows.Scan(
			&l.ID, &l.EmployeeID, &l.Type, &l.StartDate, &l.EndDate, &l.DaysCount, &reason, &l.Status,
			&l.CreatedAt, &l.UpdatedAt, &l.EmployeeName, &l.EmployeeNumber, &total,
		); err != nil {
			return nil, 0, err
		}
		if reason != nil {
			l.Reason = *reason
		}
		items = append(items, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PostgresRepository) Cancel(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE leaves SET status = $1 WHERE id = $2 AND status = $3`,
		StatusCancelled, id, StatusActive)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Distinguish "doesn't exist" from "exists but already cancelled" so
		// the handler can return the right message.
		if _, err := r.FindByID(ctx, id); err != nil {
			return err
		}
		return ErrAlreadyCancelled
	}
	return nil
}

func (r *PostgresRepository) SumActiveDays(ctx context.Context, employeeID uuid.UUID, year int) (int, error) {
	const q = `
		SELECT COALESCE(SUM(days_count), 0)
		FROM leaves
		WHERE employee_id = $1 AND status = $2 AND leave_type = $3 AND EXTRACT(YEAR FROM start_date) = $4`

	var sum int
	err := r.db.QueryRow(ctx, q, employeeID, StatusActive, TypeCuti, year).Scan(&sum)
	return sum, err
}

func (r *PostgresRepository) HasActiveOverlap(ctx context.Context, employeeID uuid.UUID, start, end time.Time) (bool, error) {
	const q = `
		SELECT EXISTS (
			SELECT 1 FROM leaves
			WHERE employee_id = $1 AND status = $2 AND start_date <= $4 AND end_date >= $3
		)`

	var exists bool
	err := r.db.QueryRow(ctx, q, employeeID, StatusActive, start, end).Scan(&exists)
	return exists, err
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

var _ Repository = (*PostgresRepository)(nil)
