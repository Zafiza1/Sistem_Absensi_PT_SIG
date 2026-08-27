package shift

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

// fakeRepository is a minimal in-memory Repository for unit-testing Service
// without a real database.
type fakeRepository struct {
	created []Shift
}

func (f *fakeRepository) Create(_ context.Context, s *Shift) error {
	s.ID = uuid.New()
	f.created = append(f.created, *s)
	return nil
}
func (f *fakeRepository) FindByID(_ context.Context, id uuid.UUID) (*Shift, error) {
	for i := range f.created {
		if f.created[i].ID == id {
			return &f.created[i], nil
		}
	}
	return nil, ErrNotFound
}
func (f *fakeRepository) List(_ context.Context, _ pagination.Params) ([]Shift, int64, error) {
	return f.created, int64(len(f.created)), nil
}
func (f *fakeRepository) Update(_ context.Context, s *Shift) error    { return nil }
func (f *fakeRepository) Delete(_ context.Context, _ uuid.UUID) error { return nil }

var _ Repository = (*fakeRepository)(nil)

func TestService_Create_DetectsOvernightShift(t *testing.T) {
	svc := NewService(&fakeRepository{})

	sh, err := svc.Create(context.Background(), Input{
		Name:                   "Shift Malam",
		StartTime:              "22:00",
		EndTime:                "06:00",
		LateToleranceMinutes:   10,
		WorkingDurationMinutes: 480,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !sh.IsOvernight {
		t.Error("IsOvernight = false, want true for a 22:00 -> 06:00 shift")
	}
}

func TestService_Create_DaytimeShiftIsNotOvernight(t *testing.T) {
	svc := NewService(&fakeRepository{})

	sh, err := svc.Create(context.Background(), Input{
		Name:                   "Shift Pagi",
		StartTime:              "07:00",
		EndTime:                "15:00",
		LateToleranceMinutes:   10,
		WorkingDurationMinutes: 480,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if sh.IsOvernight {
		t.Error("IsOvernight = true, want false for a 07:00 -> 15:00 shift")
	}
}

func TestService_Create_InvalidTimeFormat(t *testing.T) {
	svc := NewService(&fakeRepository{})

	_, err := svc.Create(context.Background(), Input{
		Name:                   "Shift Aneh",
		StartTime:              "7am",
		EndTime:                "15:00",
		WorkingDurationMinutes: 480,
	})
	if !errors.Is(err, ErrInvalidTimeFormat) {
		t.Fatalf("Create() error = %v, want ErrInvalidTimeFormat", err)
	}
}
