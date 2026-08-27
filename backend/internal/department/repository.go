package department

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
	ErrNotFound      = errors.New("department: not found")
	ErrNameTaken     = errors.New("department: name already in use")
	ErrHasReferences = errors.New("department: still referenced by other records")
)

type Repository interface {
	Create(ctx context.Context, d *Department) error
	FindByID(ctx context.Context, id uuid.UUID) (*Department, error)
	List(ctx context.Context, p pagination.Params) ([]Department, int64, error)
	Update(ctx context.Context, d *Department) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, d *Department) error {
	const q = `
		INSERT INTO departments (name, description, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, q, d.Name, d.Description, d.IsActive).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if dberr.IsUniqueViolation(err) {
		return ErrNameTaken
	}
	return err
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Department, error) {
	const q = `
		SELECT id, name, description, is_active, created_at, updated_at
		FROM departments
		WHERE id = $1`

	var d Department
	err := r.db.QueryRow(ctx, q, id).Scan(&d.ID, &d.Name, &d.Description, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *PostgresRepository) List(ctx context.Context, p pagination.Params) ([]Department, int64, error) {
	const q = `
		SELECT id, name, description, is_active, created_at, updated_at, COUNT(*) OVER() AS total
		FROM departments
		ORDER BY name
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, q, p.PageSize, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		items []Department
		total int64
	)
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.IsActive, &d.CreatedAt, &d.UpdatedAt, &total); err != nil {
			return nil, 0, err
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, d *Department) error {
	const q = `
		UPDATE departments
		SET name = $1, description = $2, is_active = $3
		WHERE id = $4
		RETURNING updated_at`

	err := r.db.QueryRow(ctx, q, d.Name, d.Description, d.IsActive, d.ID).Scan(&d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if dberr.IsUniqueViolation(err) {
		return ErrNameTaken
	}
	return err
}

// Delete removes a department, but refuses when active employees are still
// assigned to it. employees.department_id is ON DELETE SET NULL (deleting a
// department doesn't cascade or hard-fail at the FK level), so this check
// is done explicitly here to force an admin to reassign those employees
// first instead of silently orphaning their department reference.
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `
		DELETE FROM departments
		WHERE id = $1
		  AND NOT EXISTS (
		    SELECT 1 FROM employees
		    WHERE employees.department_id = $1 AND employees.deleted_at IS NULL
		  )`

	tag, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	if _, findErr := r.FindByID(ctx, id); findErr != nil {
		return findErr // ErrNotFound
	}
	return ErrHasReferences
}

var _ Repository = (*PostgresRepository)(nil)
