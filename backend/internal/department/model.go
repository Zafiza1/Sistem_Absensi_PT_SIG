// Package department manages organizational departments, the top level of
// PT Surya Inti Gas's employee grouping (Employee -> Department).
package department

import (
	"time"

	"github.com/google/uuid"
)

type Department struct {
	ID          uuid.UUID
	Name        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
