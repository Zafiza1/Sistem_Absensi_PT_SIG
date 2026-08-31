package employee

import (
	"context"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/auditlog"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

// Input bundles the caller-supplied, mutable fields of an employee.
type Input struct {
	EmployeeNumber string
	Name           string
	Email          *string
	Phone          *string
	DepartmentID   *uuid.UUID
	PositionID     *uuid.UUID
	ShiftID        *uuid.UUID
	Status         string
}

// Actor identifies who is performing a mutation, for the audit trail. It
// lives in package auditlog (shared with user and device) — see
// auditlog.Actor's doc comment.
type Actor = auditlog.Actor

type Service struct {
	repo  Repository
	audit *auditlog.Service
}

func NewService(repo Repository, audit *auditlog.Service) *Service {
	return &Service{repo: repo, audit: audit}
}

func (s *Service) Create(ctx context.Context, actor Actor, in Input) (*Employee, error) {
	status := in.Status
	if status == "" {
		status = StatusActive
	}
	e := &Employee{
		EmployeeNumber: in.EmployeeNumber,
		Name:           in.Name,
		Email:          in.Email,
		Phone:          in.Phone,
		DepartmentID:   in.DepartmentID,
		PositionID:     in.PositionID,
		ShiftID:        in.ShiftID,
		Status:         status,
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionCreate, "employee", e.ID.String(),
		"Menambahkan karyawan "+e.Name+" ("+e.EmployeeNumber+")", actor.IP)
	return e, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Employee, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, f Filter, p pagination.Params) ([]Employee, int64, error) {
	return s.repo.List(ctx, f, p)
}

func (s *Service) Update(ctx context.Context, actor Actor, id uuid.UUID, in Input) (*Employee, error) {
	status := in.Status
	if status == "" {
		status = StatusActive
	}
	e := &Employee{
		ID:             id,
		EmployeeNumber: in.EmployeeNumber,
		Name:           in.Name,
		Email:          in.Email,
		Phone:          in.Phone,
		DepartmentID:   in.DepartmentID,
		PositionID:     in.PositionID,
		ShiftID:        in.ShiftID,
		Status:         status,
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionUpdate, "employee", e.ID.String(),
		"Memperbarui karyawan "+e.Name+" ("+e.EmployeeNumber+")", actor.IP)
	return e, nil
}

// Deactivate soft-deletes the employee. "Menghapus" an employee in the
// dashboard never hard-deletes: attendance history (Phase 4) references
// employees, and losing that history when someone leaves the company would
// break reporting.
func (s *Service) Deactivate(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return err
	}

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionDelete, "employee", id.String(),
		"Menonaktifkan karyawan", actor.IP)
	return nil
}
