package position

import (
	"context"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, name, description string) (*Position, error) {
	p := &Position{Name: name, Description: description, IsActive: true}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Position, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, p pagination.Params) ([]Position, int64, error) {
	return s.repo.List(ctx, p)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, name, description string, isActive bool) (*Position, error) {
	p := &Position{ID: id, Name: name, Description: description, IsActive: isActive}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
