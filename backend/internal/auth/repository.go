package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUserNotFound and ErrRefreshTokenNotFound are returned by Repository
// implementations; Service translates them into the caller-facing
// ErrInvalidCredentials / ErrInvalidRefreshToken so a lookup miss never
// leaks "which part was wrong" to an API client.
var (
	ErrUserNotFound         = errors.New("auth: user not found")
	ErrRefreshTokenNotFound = errors.New("auth: refresh token not found")
)

// Repository is the persistence boundary for the auth module. Defined as an
// interface so Service can be unit-tested with a fake, without a real
// database.
type Repository interface {
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	FindUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	CreateRefreshToken(ctx context.Context, rt *RefreshToken) error
	FindRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
}

// PostgresRepository is the production Repository implementation.
type PostgresRepository struct {
	db *pgxpool.Pool
}

// NewPostgresRepository builds a Repository backed by the given pool.
func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	const q = `
		SELECT id, name, email, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE email = $1`

	var u User
	err := r.db.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *PostgresRepository) FindUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const q = `
		SELECT id, name, email, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1`

	var u User
	err := r.db.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *PostgresRepository) CreateRefreshToken(ctx context.Context, rt *RefreshToken) error {
	const q = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`

	if rt.ID == uuid.Nil {
		rt.ID = uuid.New()
	}
	return r.db.QueryRow(ctx, q, rt.ID, rt.UserID, rt.TokenHash, rt.ExpiresAt, rt.UserAgent, rt.IPAddress).
		Scan(&rt.CreatedAt)
}

func (r *PostgresRepository) FindRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	const q = `
		SELECT id, user_id, token_hash, expires_at, revoked_at, user_agent, ip_address, created_at
		FROM refresh_tokens
		WHERE token_hash = $1`

	var rt RefreshToken
	err := r.db.QueryRow(ctx, q, hash).Scan(
		&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.RevokedAt, &rt.UserAgent, &rt.IPAddress, &rt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, err
	}
	return &rt, nil
}

func (r *PostgresRepository) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`
	_, err := r.db.Exec(ctx, q, id)
	return err
}

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)
