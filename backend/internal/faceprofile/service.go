package faceprofile

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrEmptyVector = errors.New("faceprofile: feature vector must not be empty")

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Enroll(ctx context.Context, employeeID uuid.UUID, vector []float64) (*FaceProfile, error) {
	if len(vector) == 0 {
		return nil, ErrEmptyVector
	}
	if _, err := s.repo.Upsert(ctx, employeeID, vector); err != nil {
		return nil, err
	}
	// Upsert's RETURNING only covers face_profiles' own columns; reload
	// through the employee JOIN so the response carries employee_name/
	// employee_number like every other read in this package.
	return s.repo.FindByEmployeeID(ctx, employeeID)
}

func (s *Service) Get(ctx context.Context, employeeID uuid.UUID) (*FaceProfile, error) {
	return s.repo.FindByEmployeeID(ctx, employeeID)
}

// ListActive is what a registered tablet syncs down to run recognition
// entirely on-device.
func (s *Service) ListActive(ctx context.Context) ([]FaceProfile, error) {
	return s.repo.ListActive(ctx)
}
