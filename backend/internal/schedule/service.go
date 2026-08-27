package schedule

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

var ErrInvalidDayOfWeek = errors.New("schedule: day_of_week must be between 1 (Monday) and 7 (Sunday)")

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, employeeID, shiftID uuid.UUID, dayOfWeek int) (*Schedule, error) {
	if dayOfWeek < 1 || dayOfWeek > 7 {
		return nil, ErrInvalidDayOfWeek
	}
	sc := &Schedule{EmployeeID: employeeID, ShiftID: shiftID, DayOfWeek: dayOfWeek}
	if err := s.repo.Create(ctx, sc); err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Schedule, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) ListByEmployee(ctx context.Context, employeeID uuid.UUID) ([]Schedule, error) {
	return s.repo.ListByEmployee(ctx, employeeID)
}

func (s *Service) List(ctx context.Context, p pagination.Params) ([]Schedule, int64, error) {
	return s.repo.List(ctx, p)
}

func (s *Service) Update(ctx context.Context, id, shiftID uuid.UUID, dayOfWeek int) (*Schedule, error) {
	if dayOfWeek < 1 || dayOfWeek > 7 {
		return nil, ErrInvalidDayOfWeek
	}
	sc := &Schedule{ID: id, ShiftID: shiftID, DayOfWeek: dayOfWeek}
	if err := s.repo.Update(ctx, sc); err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
