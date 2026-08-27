package device

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suryaintigas/absensi-backend/pkg/dberr"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

var (
	ErrNotFound       = errors.New("device: not found")
	ErrDeviceCodeUsed = errors.New("device: device code already registered")
)

type Repository interface {
	Create(ctx context.Context, d *Device) error
	FindByID(ctx context.Context, id uuid.UUID) (*Device, error)
	// FindByCode looks up a device by its human/hardware-assigned
	// device_code — what the tablet app itself knows and sends on every
	// attendance request, as opposed to the internal UUID.
	FindByCode(ctx context.Context, code string) (*Device, error)
	List(ctx context.Context, p pagination.Params) ([]Device, int64, error)
	Update(ctx context.Context, d *Device) error
	Delete(ctx context.Context, id uuid.UUID) error
	// Touch records that a device was just seen (e.g. it just submitted an
	// attendance), used to derive Device.IsOnline. Best-effort: callers
	// should not fail the request that triggered it if Touch fails.
	Touch(ctx context.Context, id uuid.UUID, seenAt time.Time) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const selectColumns = `id, device_name, device_code, location, status, app_version, last_seen_at, last_sync_at, created_at, updated_at`

func scanDevice(row pgx.Row, d *Device) error {
	return row.Scan(&d.ID, &d.DeviceName, &d.DeviceCode, &d.Location, &d.Status, &d.AppVersion,
		&d.LastSeenAt, &d.LastSyncAt, &d.CreatedAt, &d.UpdatedAt)
}

func (r *PostgresRepository) Create(ctx context.Context, d *Device) error {
	const q = `
		INSERT INTO devices (device_name, device_code, location, status, app_version)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, q, d.DeviceName, d.DeviceCode, d.Location, d.Status, d.AppVersion).
		Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if dberr.IsUniqueViolation(err) {
		return ErrDeviceCodeUsed
	}
	return err
}

func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*Device, error) {
	q := `SELECT ` + selectColumns + ` FROM devices WHERE id = $1`

	var d Device
	err := scanDevice(r.db.QueryRow(ctx, q, id), &d)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *PostgresRepository) FindByCode(ctx context.Context, code string) (*Device, error) {
	q := `SELECT ` + selectColumns + ` FROM devices WHERE device_code = $1`

	var d Device
	err := scanDevice(r.db.QueryRow(ctx, q, code), &d)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *PostgresRepository) Touch(ctx context.Context, id uuid.UUID, seenAt time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE devices SET last_seen_at = $1 WHERE id = $2`, seenAt, id)
	return err
}

func (r *PostgresRepository) List(ctx context.Context, p pagination.Params) ([]Device, int64, error) {
	q := `SELECT ` + selectColumns + `, COUNT(*) OVER() AS total FROM devices ORDER BY device_name LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, q, p.PageSize, p.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		items []Device
		total int64
	)
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.DeviceName, &d.DeviceCode, &d.Location, &d.Status, &d.AppVersion,
			&d.LastSeenAt, &d.LastSyncAt, &d.CreatedAt, &d.UpdatedAt, &total); err != nil {
			return nil, 0, err
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, d *Device) error {
	const q = `
		UPDATE devices
		SET device_name = $1, device_code = $2, location = $3, status = $4, app_version = $5
		WHERE id = $6
		RETURNING updated_at`

	err := r.db.QueryRow(ctx, q, d.DeviceName, d.DeviceCode, d.Location, d.Status, d.AppVersion, d.ID).
		Scan(&d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if dberr.IsUniqueViolation(err) {
		return ErrDeviceCodeUsed
	}
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM devices WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

var _ Repository = (*PostgresRepository)(nil)
