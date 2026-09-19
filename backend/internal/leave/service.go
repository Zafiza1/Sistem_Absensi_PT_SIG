package leave

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/auditlog"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

// ErrQuotaExceeded is returned by Request when granting the requested
// period would push the employee's ACTIVE days for that calendar year past
// AnnualQuotaDays.
var ErrQuotaExceeded = errors.New("leave: annual leave quota exceeded")

// Input bundles the caller-supplied fields of a leave request.
type Input struct {
	EmployeeID uuid.UUID
	StartDate  time.Time
	EndDate    time.Time
	Reason     string
}

// Actor identifies who is performing a mutation, for the audit trail.
type Actor = auditlog.Actor

type Service struct {
	repo  Repository
	audit *auditlog.Service
}

func NewService(repo Repository, audit *auditlog.Service) *Service {
	return &Service{repo: repo, audit: audit}
}

// Request records a new leave period, after checking it fits within the
// employee's remaining annual quota (AnnualQuotaDays per calendar year, per
// company policy). start and end are inclusive calendar dates, so a single
// day off has start == end and DaysCount == 1.
func (s *Service) Request(ctx context.Context, actor Actor, in Input) (*Leave, error) {
	days := int(in.EndDate.Sub(in.StartDate).Hours()/24) + 1

	used, err := s.repo.SumActiveDays(ctx, in.EmployeeID, in.StartDate.Year())
	if err != nil {
		return nil, err
	}
	if used+days > AnnualQuotaDays {
		return nil, ErrQuotaExceeded
	}

	l := &Leave{
		EmployeeID: in.EmployeeID,
		StartDate:  in.StartDate,
		EndDate:    in.EndDate,
		DaysCount:  days,
		Reason:     in.Reason,
		Status:     StatusActive,
	}
	if err := s.repo.Create(ctx, l); err != nil {
		return nil, err
	}

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionCreate, "leave", l.ID.String(),
		"Mencatat cuti "+l.EmployeeName+" ("+l.StartDate.Format("2006-01-02")+" s/d "+l.EndDate.Format("2006-01-02")+")", actor.IP)
	return l, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Leave, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, f Filter, p pagination.Params) ([]Leave, int64, error) {
	return s.repo.List(ctx, f, p)
}

// Cancel frees the leave's days back up against the employee's annual
// quota without deleting the record.
func (s *Service) Cancel(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := s.repo.Cancel(ctx, id); err != nil {
		return err
	}

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionDelete, "leave", id.String(),
		"Membatalkan cuti", actor.IP)
	return nil
}

// Balance reports how many of the employee's AnnualQuotaDays for year
// remain unused.
func (s *Service) Balance(ctx context.Context, employeeID uuid.UUID, year int) (used int, remaining int, err error) {
	used, err = s.repo.SumActiveDays(ctx, employeeID, year)
	if err != nil {
		return 0, 0, err
	}
	remaining = AnnualQuotaDays - used
	if remaining < 0 {
		remaining = 0
	}
	return used, remaining, nil
}
