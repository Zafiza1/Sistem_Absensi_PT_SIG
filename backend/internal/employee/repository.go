package employee

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suryaintigas/absensi-backend/pkg/dberr"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

var (
	ErrNotFound            = errors.New("employee: not found")
	ErrEmployeeNumberTaken = errors.New("employee: employee number already in use")
	ErrEmailTaken          = errors.New("employee: email already in use")
	ErrInvalidReference    = errors.New("employee: department, position, or shift not found")
)

// Filter narrows List() results. Zero-value fields are ignored.
type Filter struct {
	Search       string // matches name or employee_number, case-insensitive
	DepartmentID *uuid.UUID
	Status       string
}

type Repository interface {
	Create(ctx context.Context, e *Employee) error
	FindByID(ctx context.Context, id uuid.UUID) (*Employee, error)
	List(ctx context.Context, f Filter, p pagination.Params) ([]Employee, int64, error)
	Update(ctx context.Context, e *Employee) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// employeeColumns and employeeFrom are kept separate (rather than one
// combined constant) so List() can splice ", COUNT(*) OVER() AS total"
// into the SELECT column list — inserting it after a combined
// "SELECT ... FROM ... JOIN ..." string would land it after the JOINs
// instead, producing invalid SQL.
const employeeColumns = `
	e.id, e.employee_number, e.name, e.email, e.phone,
	e.department_id, e.position_id, e.shift_id, e.status,
	e.created_at, e.updated_at, e.deleted_at,
	COALESCE(d.name, ''), COALESCE(p.name, ''), COALESCE(s.name, '')`

const employeeFrom = `
	FROM employees e
	LEFT JOIN departments d ON d.id = e.department_id
	LEFT JOIN positions p ON p.id = e.position_id
	LEFT JOIN shifts s ON s.id = e.shift_id`

const baseSelect = "SELECT " + employeeColumns + employeeFrom

func scanEmployee(row pgx.Row, e *Employee) error {
	return row.Scan(
		&e.ID, &e.EmployeeNumber, &e.Name, &e.Email, &e.Phone,
		&e.DepartmentID, &e.PositionID, &e.ShiftID, &e.Status,
		&e.CreatedAt, &e.UpdatedAt, &e.DeletedAt,
		&e.DepartmentName, &e.PositionName, &e.ShiftName,
	)
}

func (r *PostgresRepository) Create(ctx context.Context, e *Employee) error {
	const q = `
		INSERT INTO employees (employee_number, name, email, phone, department_id, position_id, shift_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, q, e.EmployeeNumber, e.Name, e.Email, e.Phone, e.DepartmentID, e.PositionID, e.ShiftID, e.Status).
		Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if dberr.IsUniqueViolation(err) {
		return classifyUniqueViolation(err)
	}
	if dberr.IsForeignKeyViolation(err) {
		return ErrInvalidReference
	}
	if err != nil {
		return err
	}
	return r.reload(ctx, e)
}

// reload re-fetches e by ID and overwrites it in place, populating the
// department/position/shift display names joined in from other tables —
// the plain INSERT/UPDATE ... RETURNING above only has the raw employees
// row available, not those joins.
func (r *PostgresRepository) reload(ctx context.Context, e *Employee) error {
	full, err := r.FindByID(ctx, e.ID)
	if err != nil {
		return err
	}
	*e = *full
	return nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Employee, error) {
	q := baseSelect + ` WHERE e.id = $1 AND e.deleted_at IS NULL`

	var e Employee
	err := scanEmployee(r.db.QueryRow(ctx, q, id), &e)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *PostgresRepository) List(ctx context.Context, f Filter, p pagination.Params) ([]Employee, int64, error) {
	var (
		conditions = []string{"e.deleted_at IS NULL"}
		args       []any
	)
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	if f.Search != "" {
		placeholder := arg("%" + strings.ToLower(f.Search) + "%")
		conditions = append(conditions, "(LOWER(e.name) LIKE "+placeholder+" OR LOWER(e.employee_number) LIKE "+placeholder+")")
	}
	if f.DepartmentID != nil {
		conditions = append(conditions, "e.department_id = "+arg(*f.DepartmentID))
	}
	if f.Status != "" {
		conditions = append(conditions, "e.status = "+arg(f.Status))
	}

	q := "SELECT " + employeeColumns + ", COUNT(*) OVER() AS total" + employeeFrom +
		" WHERE " + strings.Join(conditions, " AND ") +
		" ORDER BY e.name LIMIT " + arg(p.PageSize) + " OFFSET " + arg(p.Offset())

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		items []Employee
		total int64
	)
	for rows.Next() {
		var e Employee
		if err := rows.Scan(
			&e.ID, &e.EmployeeNumber, &e.Name, &e.Email, &e.Phone,
			&e.DepartmentID, &e.PositionID, &e.ShiftID, &e.Status,
			&e.CreatedAt, &e.UpdatedAt, &e.DeletedAt,
			&e.DepartmentName, &e.PositionName, &e.ShiftName, &total,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, e *Employee) error {
	const q = `
		UPDATE employees
		SET employee_number = $1, name = $2, email = $3, phone = $4,
		    department_id = $5, position_id = $6, shift_id = $7, status = $8
		WHERE id = $9 AND deleted_at IS NULL
		RETURNING updated_at`

	err := r.db.QueryRow(ctx, q, e.EmployeeNumber, e.Name, e.Email, e.Phone,
		e.DepartmentID, e.PositionID, e.ShiftID, e.Status, e.ID).
		Scan(&e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if dberr.IsUniqueViolation(err) {
		return classifyUniqueViolation(err)
	}
	if dberr.IsForeignKeyViolation(err) {
		return ErrInvalidReference
	}
	if err != nil {
		return err
	}
	return r.reload(ctx, e)
}

func (r *PostgresRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `UPDATE employees SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// classifyUniqueViolation inspects which partial unique index tripped so
// the API can return a field-specific message instead of a generic one.
func classifyUniqueViolation(err error) error {
	if strings.Contains(err.Error(), "employee_number") {
		return ErrEmployeeNumberTaken
	}
	if strings.Contains(err.Error(), "email") {
		return ErrEmailTaken
	}
	return ErrEmployeeNumberTaken
}

var _ Repository = (*PostgresRepository)(nil)
