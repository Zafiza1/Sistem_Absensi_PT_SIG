package companyschedule

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/auditlog"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

// fakeAuditRepo is a no-op auditlog.Repository so Service.Set's audit call
// has something to write to without a database.
type fakeAuditRepo struct{ recorded int }

func (f *fakeAuditRepo) Record(context.Context, *auditlog.Entry) error { f.recorded++; return nil }
func (f *fakeAuditRepo) List(context.Context, auditlog.Filter, pagination.Params) ([]auditlog.Entry, int64, error) {
	return nil, 0, nil
}

type fakeRepo struct {
	days      []Day
	replaceIn []Day
	failShift *uuid.UUID // Replace returns ErrInvalidShift if any day uses this shift
}

func (f *fakeRepo) List(context.Context) ([]Day, error) { return f.days, nil }
func (f *fakeRepo) FindByDay(_ context.Context, day int) (*Day, error) {
	for i := range f.days {
		if f.days[i].DayOfWeek == day {
			return &f.days[i], nil
		}
	}
	return nil, ErrNotFound
}
func (f *fakeRepo) Replace(_ context.Context, days []Day) error {
	if f.failShift != nil {
		for _, d := range days {
			if d.ShiftID != nil && *d.ShiftID == *f.failShift {
				return ErrInvalidShift
			}
		}
	}
	f.replaceIn = days
	f.days = days
	return nil
}

func newService(repo Repository) *Service {
	return NewService(repo, auditlog.NewService(&fakeAuditRepo{}))
}

func TestService_Set_ReplacesWholeWeek(t *testing.T) {
	shiftA := uuid.New()
	repo := &fakeRepo{}
	svc := newService(repo)

	_, err := svc.Set(context.Background(), Actor{}, []DayInput{
		{DayOfWeek: 1, ShiftID: &shiftA},
		{DayOfWeek: 6, ShiftID: &shiftA},
		{DayOfWeek: 7, ShiftID: nil}, // Sunday off
	})
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if len(repo.replaceIn) != 3 {
		t.Fatalf("Replace got %d days, want 3", len(repo.replaceIn))
	}
}

func TestService_Set_RejectsBadDayOfWeek(t *testing.T) {
	svc := newService(&fakeRepo{})
	_, err := svc.Set(context.Background(), Actor{}, []DayInput{{DayOfWeek: 8}})
	if !errors.Is(err, ErrInvalidDay) {
		t.Fatalf("Set() error = %v, want ErrInvalidDay", err)
	}
}

func TestService_Set_RejectsDuplicateDay(t *testing.T) {
	svc := newService(&fakeRepo{})
	_, err := svc.Set(context.Background(), Actor{}, []DayInput{
		{DayOfWeek: 1}, {DayOfWeek: 1},
	})
	if !errors.Is(err, ErrDuplicateDay) {
		t.Fatalf("Set() error = %v, want ErrDuplicateDay", err)
	}
}

func TestService_Set_PropagatesUnknownShift(t *testing.T) {
	bad := uuid.New()
	svc := newService(&fakeRepo{failShift: &bad})
	_, err := svc.Set(context.Background(), Actor{}, []DayInput{{DayOfWeek: 1, ShiftID: &bad}})
	if !errors.Is(err, ErrInvalidShift) {
		t.Fatalf("Set() error = %v, want ErrInvalidShift", err)
	}
}
