package user

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
	ErrNotFound   = errors.New("user: not found")
	ErrEmailTaken = errors.New("user: email already in use")
)

type Repository interface {
	Create(ctx context.Context, u *User, passwordHash string) error
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	List(ctx context.Context, p pagination.Params) ([]User, int64, error)
	Update(ctx context.Context, u *User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, u *User, passwordHash string) error {
	const q = `
		INSERT INTO users (name, email, password_hash, role, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, q, u.Name, u.Email, passwordHash, u.Role, u.IsActive).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if dberr.IsUniqueViolation(err) {
		return ErrEmailTaken
	}
	return err
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const q = `
		SELECT id, name, email, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1`

	var u User
	err := r.db.QueryRow(ctx, q, id).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *PostgresRepository) List(ctx context.Context, p pagination.Params) ([]User, int64, error) {
	const q = `
		SELECT id, name, email, role, is_active, created_at, updated_at, COUNT(*) OVER() AS total
		FROM users
		ORDER BY name
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, q, p.PageSize, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		items []User
		total int64
	)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &total); err != nil {
			return nil, 0, err
		}
		items = append(items, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, u *User) error {
	const q = `
		UPDATE users
		SET name = $1, email = $2, role = $3, is_active = $4
		WHERE id = $5
		RETURNING updated_at`

	err := r.db.QueryRow(ctx, q, u.Name, u.Email, u.Role, u.IsActive, u.ID).Scan(&u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if dberr.IsUniqueViolation(err) {
		return ErrEmailTaken
	}
	return err
}

func (r *PostgresRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	const q = `UPDATE users SET password_hash = $1 WHERE id = $2`
	tag, err := r.db.Exec(ctx, q, passwordHash, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete deactivates rather than removes the row outright — same
// soft-delete-by-status pattern as Employees and Devices (Phase 3). A
// dashboard account that created master data, attendance corrections, or
// audit-log entries shouldn't vanish and orphan those references; it just
// loses the ability to log in (auth.Service.Login already refuses inactive
// accounts).
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE users SET is_active = false WHERE id = $1`
	tag, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

var _ Repository = (*PostgresRepository)(nil)
