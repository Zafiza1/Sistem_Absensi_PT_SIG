package auditlog

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

type Repository interface {
	Record(ctx context.Context, e *Entry) error
	List(ctx context.Context, f Filter, p pagination.Params) ([]Entry, int64, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Record(ctx context.Context, e *Entry) error {
	const q = `
		INSERT INTO audit_logs (actor_id, actor_name, actor_role, action, entity_type, entity_id, description, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`

	var entityID *string
	if e.EntityID != "" {
		entityID = &e.EntityID
	}

	return r.db.QueryRow(ctx, q,
		e.ActorID, e.ActorName, e.ActorRole, e.Action, e.EntityType, entityID, e.Description, e.IPAddress,
	).Scan(&e.ID, &e.CreatedAt)
}

func (r *PostgresRepository) List(ctx context.Context, f Filter, p pagination.Params) ([]Entry, int64, error) {
	const q = `
		SELECT id, actor_id, actor_name, actor_role, action, entity_type, COALESCE(entity_id, ''),
		       description, COALESCE(ip_address, ''), created_at, COUNT(*) OVER() AS total
		FROM audit_logs
		WHERE ($1::uuid IS NULL OR actor_id = $1)
		  AND ($2::text = '' OR entity_type = $2)
		  AND ($3::text = '' OR action = $3)
		  AND ($4::timestamptz IS NULL OR created_at >= $4)
		  AND ($5::timestamptz IS NULL OR created_at <= $5)
		ORDER BY created_at DESC
		LIMIT $6 OFFSET $7`

	rows, err := r.db.Query(ctx, q,
		f.ActorID, f.EntityType, string(f.Action), f.DateFrom, f.DateTo, p.PageSize, p.Offset(),
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		items []Entry
		total int64
	)
	for rows.Next() {
		var e Entry
		if err := rows.Scan(
			&e.ID, &e.ActorID, &e.ActorName, &e.ActorRole, &e.Action, &e.EntityType,
			&e.EntityID, &e.Description, &e.IPAddress, &e.CreatedAt, &total,
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

var _ Repository = (*PostgresRepository)(nil)
