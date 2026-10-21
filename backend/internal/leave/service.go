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

// ErrInvalidType is returned by Request for a type outside TypeCuti,
// TypeIzin and TypeSakit.
var ErrInvalidType = errors.New("leave: invalid leave type")

// Input bundles the caller-supplied fields of a leave request.
type Input struct {
	EmployeeID uuid.UUID
	Type       string // TypeCuti, TypeIzin or TypeSakit; empty means TypeCuti
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

// Request records a new leave period. A cuti must fit within the
// employee's remaining annual quota (AnnualQuotaDays per calendar year, per
// company policy); izin and sakit have no quota. No type may overlap an
// already-active leave, so a day is never counted twice. start and end are
// inclusive calendar dates, so a single day off has start == end and
// DaysCount == 1.
func (s *Service) Request(ctx context.Context, actor Actor, in Input) (*Leave, error) {
	if in.Type == "" {
		in.Type = TypeCuti
	}
	if !ValidType(in.Type) {
		return nil, ErrInvalidType
	}
	days := int(in.EndDate.Sub(in.StartDate).Hours()/24) + 1

	overlap, err := s.repo.HasActiveOverlap(ctx, in.EmployeeID, in.StartDate, in.EndDate)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, ErrOverlap
	}

	if in.Type == TypeCuti {
		used, err := s.repo.SumActiveDays(ctx, in.EmployeeID, in.StartDate.Year())
		if err != nil {
			return nil, err
		}
		if used+days > AnnualQuotaDays {
			return nil, ErrQuotaExceeded
		}
	}

	l := &Leave{
		EmployeeID: in.EmployeeID,
		Type:       in.Type,
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
		"Mencatat "+TypeLabel(l.Type)+" "+l.EmployeeName+" ("+l.StartDate.Format("2006-01-02")+" s/d "+l.EndDate.Format("2006-01-02")+")", actor.IP)
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

// TypeLabel is the Indonesian display name of a leave type.
func TypeLabel(t string) string {
	switch t {
	case TypeIzin:
		return "izin"
	case TypeSakit:
		return "sakit"
	default:
		return "cuti"
	}
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
