package device

import (
	"context"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/auditlog"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

type Input struct {
	DeviceName   string
	DeviceCode   string
	Location     string
	AppVersion   string
	DeviceType   string
	SerialNumber string
	IPAddress    string
	Port         int
	DeviceConfig map[string]interface{}
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
// Updated for Fingerspot device integration.
func (s *Service) Register(ctx context.Context, actor Actor, in Input) (*Device, error) {
	// Set default device type if not provided
	deviceType := in.DeviceType
	if deviceType == "" {
		deviceType = DeviceTypeFingerspot
	}

	// Set default connection status
	connectionStatus := ConnectionStatusDisconnected

	d := &Device{
		DeviceName:       in.DeviceName,
		DeviceCode:       in.DeviceCode,
		Location:         in.Location,
		AppVersion:       in.AppVersion,
		Status:           StatusActive,
		DeviceType:       deviceType,
		SerialNumber:     in.SerialNumber,
		IPAddress:        in.IPAddress,
		Port:             in.Port,
		ConnectionStatus: connectionStatus,
		SyncStatus:       SyncStatusIdle,
		DeviceConfig:     in.DeviceConfig,
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

	// Get existing device to preserve connection status and sync status
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	d := &Device{
		ID:               id,
		DeviceName:       in.DeviceName,
		DeviceCode:       in.DeviceCode,
		Location:         in.Location,
		AppVersion:       in.AppVersion,
		Status:           status,
		DeviceType:       in.DeviceType,
		SerialNumber:     in.SerialNumber,
		IPAddress:        in.IPAddress,
		Port:             in.Port,
		ConnectionStatus: existing.ConnectionStatus, // Preserve existing connection status
		SyncStatus:       existing.SyncStatus,       // Preserve existing sync status
		ErrorMessage:     existing.ErrorMessage,     // Preserve existing error message
		DeviceConfig:     in.DeviceConfig,
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

// UpdateStatus updates the connection and sync status of a device
func (s *Service) UpdateStatus(ctx context.Context, actor Actor, id uuid.UUID, connectionStatus, syncStatus, errorMessage string) error {
	// Use the repository method to update status fields
	if err := s.repo.UpdateSyncStatus(ctx, id, syncStatus, connectionStatus, errorMessage); err != nil {
		return err
	}

	// Get device name for audit log
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Record audit log
	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionUpdate, "device", id.String(),
		"Memperbarui status perangkat "+existing.DeviceName, actor.IP)

	return nil
}
