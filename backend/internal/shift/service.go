package shift

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

// ErrInvalidTimeFormat is returned when start/end time isn't valid "HH:MM".
var ErrInvalidTimeFormat = errors.New("shift: time must be in HH:MM format")

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Input bundles the caller-supplied fields for Create/Update — named to
// keep both method signatures short given the field count.
type Input struct {
	Name                   string
	StartTime              string
	EndTime                string
	LateToleranceMinutes   int
	WorkingDurationMinutes int
}

func (s *Service) Create(ctx context.Context, in Input) (*Shift, error) {
	if err := validateClock(in.StartTime); err != nil {
		return nil, err
	}
	if err := validateClock(in.EndTime); err != nil {
		return nil, err
	}

	sh := &Shift{
		Name:                   in.Name,
		StartTime:              in.StartTime,
		EndTime:                in.EndTime,
		IsOvernight:            isOvernight(in.StartTime, in.EndTime),
		LateToleranceMinutes:   in.LateToleranceMinutes,
		WorkingDurationMinutes: in.WorkingDurationMinutes,
		IsActive:               true,
	}
	if err := s.repo.Create(ctx, sh); err != nil {
		return nil, err
	}
	return sh, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Shift, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, p pagination.Params) ([]Shift, int64, error) {
	return s.repo.List(ctx, p)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in Input, isActive bool) (*Shift, error) {
	if err := validateClock(in.StartTime); err != nil {
		return nil, err
	}
	if err := validateClock(in.EndTime); err != nil {
		return nil, err
	}

	sh := &Shift{
		ID:                     id,
		Name:                   in.Name,
		StartTime:              in.StartTime,
		EndTime:                in.EndTime,
		IsOvernight:            isOvernight(in.StartTime, in.EndTime),
		LateToleranceMinutes:   in.LateToleranceMinutes,
		WorkingDurationMinutes: in.WorkingDurationMinutes,
		IsActive:               isActive,
	}
	if err := s.repo.Update(ctx, sh); err != nil {
		return nil, err
	}
	return sh, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func validateClock(value string) error {
	if _, err := time.Parse("15:04", value); err != nil {
		return ErrInvalidTimeFormat
	}
	return nil
}

// isOvernight reports whether a shift starting at start and ending at end
// crosses midnight (e.g. 22:00 -> 06:00). Both values are already validated
// "HH:MM" by the time this runs.
func isOvernight(start, end string) bool {
	startT, _ := time.Parse("15:04", start)
	endT, _ := time.Parse("15:04", end)
	return !endT.After(startT)
}
