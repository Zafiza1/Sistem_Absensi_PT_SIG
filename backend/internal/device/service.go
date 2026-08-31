package device

import (
	"context"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/auditlog"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

type Input struct {
	DeviceName string
	DeviceCode string
	Location   string
	AppVersion string
}

// Actor identifies who is performing a mutation, for the audit trail. It
// lives in package auditlog (shared with user and employee) — see
// auditlog.Actor's doc comment.
type Actor = auditlog.Actor

type Service struct {
	repo  Repository
	audit *auditlog.Service
}

func NewService(repo Repository, audit *auditlog.Service) *Service {
	return &Service{repo: repo, audit: audit}
}

// Register creates a new device. Named to match the spec's
// POST /api/v1/devices/register endpoint — in Phase 3 this is an
// admin-only dashboard action; a tablet self-registration flow (with its
// own device-level auth) is a Phase 5 concern.
func (s *Service) Register(ctx context.Context, actor Actor, in Input) (*Device, error) {
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

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionCreate, "device", d.ID.String(),
		"Mendaftarkan perangkat "+d.DeviceName+" ("+d.DeviceCode+")", actor.IP)
	return d, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Device, error) {
	return s.repo.FindByID(ctx, id)
}

// GetByCode backs the tablet's device-verification flow (Phase 5): the app
// looks itself up by its assigned device_code, not its internal UUID.
func (s *Service) GetByCode(ctx context.Context, code string) (*Device, error) {
	return s.repo.FindByCode(ctx, code)
}

func (s *Service) List(ctx context.Context, p pagination.Params) ([]Device, int64, error) {
	return s.repo.List(ctx, p)
}

func (s *Service) Update(ctx context.Context, actor Actor, id uuid.UUID, in Input, status string) (*Device, error) {
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

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionUpdate, "device", d.ID.String(),
		"Memperbarui perangkat "+d.DeviceName+" ("+d.DeviceCode+"), status: "+d.Status, actor.IP)
	return d, nil
}

func (s *Service) Delete(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionDelete, "device", id.String(),
		"Menghapus perangkat", actor.IP)
	return nil
}
