// Package dberr classifies PostgreSQL errors by SQLSTATE so domain services
// can turn "unique_violation" / "foreign_key_violation" into friendly,
// specific errors instead of a generic 500 with a raw database message
// leaking to the API response.
package dberr

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsUniqueViolation reports whether err is a Postgres unique_violation
// (SQLSTATE 23505) — e.g. a duplicate department name or employee number.
func IsUniqueViolation(err error) bool {
	return hasCode(err, "23505")
}

// IsForeignKeyViolation reports whether err is a Postgres
// foreign_key_violation (SQLSTATE 23503) — e.g. deleting a shift that
// active work_schedules still reference.
func IsForeignKeyViolation(err error) bool {
	return hasCode(err, "23503")
}

func hasCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == code
	}
	return false
}
