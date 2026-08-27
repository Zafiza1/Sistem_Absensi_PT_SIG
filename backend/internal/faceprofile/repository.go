package faceprofile

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suryaintigas/absensi-backend/pkg/dberr"
)

var (
	ErrNotFound        = errors.New("faceprofile: not found")
	ErrInvalidEmployee = errors.New("faceprofile: employee not found")
)

type Repository interface {
	// Upsert creates or replaces the employee's face profile — re-enrolling
	// a face (e.g. after a haircut, glasses, aging) is expected to be
	// routine, so this is idempotent rather than erroring on a repeat call.
	Upsert(ctx context.Context, employeeID uuid.UUID, vector []float64) (*FaceProfile, error)
	FindByEmployeeID(ctx context.Context, employeeID uuid.UUID) (*FaceProfile, error)
	// ListActive returns every enrolled profile belonging to a currently
	// active employee — what a registered tablet downloads to run
	// recognition entirely on-device, including while offline.
	ListActive(ctx context.Context) ([]FaceProfile, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Upsert(ctx context.Context, employeeID uuid.UUID, vector []float64) (*FaceProfile, error) {
	raw, err := json.Marshal(vector)
	if err != nil {
		return nil, err
	}

	const q = `
		INSERT INTO face_profiles (employee_id, feature_vector)
		VALUES ($1, $2)
		ON CONFLICT (employee_id) DO UPDATE
			SET feature_vector = EXCLUDED.feature_vector, updated_at = now()
		RETURNING id, created_at, updated_at`

	var fp FaceProfile
	fp.EmployeeID = employeeID
	fp.FeatureVector = vector
	err = r.db.QueryRow(ctx, q, employeeID, raw).Scan(&fp.ID, &fp.CreatedAt, &fp.UpdatedAt)
	if dberr.IsForeignKeyViolation(err) {
		return nil, ErrInvalidEmployee
	}
	if err != nil {
		return nil, err
	}
	return &fp, nil
}

func (r *PostgresRepository) FindByEmployeeID(ctx context.Context, employeeID uuid.UUID) (*FaceProfile, error) {
	const q = `
		SELECT fp.id, fp.employee_id, fp.feature_vector, fp.created_at, fp.updated_at, e.name, e.employee_number
		FROM face_profiles fp
		JOIN employees e ON e.id = fp.employee_id
		WHERE fp.employee_id = $1`

	var fp FaceProfile
	var raw []byte
	err := r.db.QueryRow(ctx, q, employeeID).Scan(&fp.ID, &fp.EmployeeID, &raw, &fp.CreatedAt, &fp.UpdatedAt, &fp.EmployeeName, &fp.EmployeeNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &fp.FeatureVector); err != nil {
		return nil, err
	}
	return &fp, nil
}

func (r *PostgresRepository) ListActive(ctx context.Context) ([]FaceProfile, error) {
	const q = `
		SELECT fp.id, fp.employee_id, fp.feature_vector, fp.created_at, fp.updated_at, e.name, e.employee_number
		FROM face_profiles fp
		JOIN employees e ON e.id = fp.employee_id
		WHERE e.status = 'ACTIVE' AND e.deleted_at IS NULL
		ORDER BY e.name`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []FaceProfile
	for rows.Next() {
		var fp FaceProfile
		var raw []byte
		if err := rows.Scan(&fp.ID, &fp.EmployeeID, &raw, &fp.CreatedAt, &fp.UpdatedAt, &fp.EmployeeName, &fp.EmployeeNumber); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &fp.FeatureVector); err != nil {
			return nil, err
		}
		items = append(items, fp)
	}
	return items, rows.Err()
}

var _ Repository = (*PostgresRepository)(nil)
