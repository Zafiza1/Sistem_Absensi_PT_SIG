package device

import (
	"context"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

type Input struct {
	DeviceName string
	DeviceCode string
	Location   string
	AppVersion string
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Register creates a new device. Named to match the spec's
// POST /api/v1/devices/register endpoint — in Phase 3 this is an
// admin-only dashboard action; a tablet self-registration flow (with its
// own device-level auth) is a Phase 5 concern.
func (s *Service) Register(ctx context.Context, in Input) (*Device, error) {
	d := &Device{
		DeviceName: in.DeviceName,
		DeviceCode: in.DeviceCode,
		Location:   in.Location,
		AppVersion: in.AppVersion,
		Status:     StatusActive,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Device, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, p pagination.Params) ([]Device, int64, error) {
	return s.repo.List(ctx, p)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in Input, status string) (*Device, error) {
	if status == "" {
		status = StatusActive
	}
	d := &Device{
		ID:         id,
		DeviceName: in.DeviceName,
		DeviceCode: in.DeviceCode,
		Location:   in.Location,
		AppVersion: in.AppVersion,
		Status:     status,
	}
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
