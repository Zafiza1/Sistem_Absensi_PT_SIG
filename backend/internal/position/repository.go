package position

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
	ErrNotFound      = errors.New("position: not found")
	ErrNameTaken     = errors.New("position: name already in use")
	ErrHasReferences = errors.New("position: still referenced by other records")
)

type Repository interface {
	Create(ctx context.Context, p *Position) error
	FindByID(ctx context.Context, id uuid.UUID) (*Position, error)
	List(ctx context.Context, p pagination.Params) ([]Position, int64, error)
	Update(ctx context.Context, p *Position) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, p *Position) error {
	const q = `
		INSERT INTO positions (name, description, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, q, p.Name, p.Description, p.IsActive).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if dberr.IsUniqueViolation(err) {
		return ErrNameTaken
	}
	return err
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Position, error) {
	const q = `
		SELECT id, name, description, is_active, created_at, updated_at
		FROM positions
		WHERE id = $1`

	var p Position
	err := r.db.QueryRow(ctx, q, id).Scan(&p.ID, &p.Name, &p.Description, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PostgresRepository) List(ctx context.Context, params pagination.Params) ([]Position, int64, error) {
	const q = `
		SELECT id, name, description, is_active, created_at, updated_at, COUNT(*) OVER() AS total
		FROM positions
		ORDER BY name
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, q, params.PageSize, params.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		items []Position
		total int64
	)
	for rows.Next() {
		var p Position
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &total); err != nil {
			return nil, 0, err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, p *Position) error {
	const q = `
		UPDATE positions
		SET name = $1, description = $2, is_active = $3
		WHERE id = $4
		RETURNING updated_at`

	err := r.db.QueryRow(ctx, q, p.Name, p.Description, p.IsActive, p.ID).Scan(&p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if dberr.IsUniqueViolation(err) {
		return ErrNameTaken
	}
	return err
}

// Delete refuses when active employees still hold this position — see
// department.Delete for why this is checked explicitly rather than relying
// on the FK (employees.position_id is ON DELETE SET NULL).
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `
		DELETE FROM positions
		WHERE id = $1
		  AND NOT EXISTS (
		    SELECT 1 FROM employees
		    WHERE employees.position_id = $1 AND employees.deleted_at IS NULL
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
