package companyschedule

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/auditlog"
)

var (
	ErrInvalidDay   = errors.New("company schedule: day_of_week must be between 1 (Monday) and 7 (Sunday)")
	ErrDuplicateDay = errors.New("company schedule: the same day_of_week appears more than once")
)

// Actor identifies who is performing a mutation, for the audit trail — see
// auditlog.Actor's doc comment for why it lives there.
type Actor = auditlog.Actor

type Service struct {
	repo  Repository
	audit *auditlog.Service
}

func NewService(repo Repository, audit *auditlog.Service) *Service {
	return &Service{repo: repo, audit: audit}
}

// DayInput is one weekday in a Set request. ShiftID nil means that weekday
// is a non-working day.
type DayInput struct {
	DayOfWeek int
	ShiftID   *uuid.UUID
}

func (s *Service) List(ctx context.Context) ([]Day, error) {
	return s.repo.List(ctx)
}

// Set replaces the entire company-wide weekly schedule with inputs. Days
// omitted from inputs become "not configured" (attendance falls back to the
// employee's default shift for them).
func (s *Service) Set(ctx context.Context, actor Actor, inputs []DayInput) ([]Day, error) {
	seen := make(map[int]bool, len(inputs))
	days := make([]Day, 0, len(inputs))
	for _, in := range inputs {
		if in.DayOfWeek < 1 || in.DayOfWeek > 7 {
			return nil, ErrInvalidDay
		}
		if seen[in.DayOfWeek] {
			return nil, ErrDuplicateDay
		}
		seen[in.DayOfWeek] = true
		days = append(days, Day{DayOfWeek: in.DayOfWeek, ShiftID: in.ShiftID})
	}

	if err := s.repo.Replace(ctx, days); err != nil {
		return nil, err
	}

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionUpdate,
		"company_schedule", "weekly", "Memperbarui jadwal kerja perusahaan", actor.IP)

	return s.repo.List(ctx)
}
