// Package position manages job positions/titles (jabatan) assignable to
// employees.
package position

import (
	"time"

	"github.com/google/uuid"
)

type Position struct {
	ID          uuid.UUID
	Name        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
