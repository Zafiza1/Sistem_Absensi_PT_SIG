package department

import (
	"context"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

// Service implements department business logic. It only depends on the
// Repository interface, so it can be unit-tested against a fake.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, name, description string) (*Department, error) {
	d := &Department{Name: name, Description: description, IsActive: true}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Department, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, p pagination.Params) ([]Department, int64, error) {
	return s.repo.List(ctx, p)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, name, description string, isActive bool) (*Department, error) {
	d := &Department{ID: id, Name: name, Description: description, IsActive: isActive}
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
