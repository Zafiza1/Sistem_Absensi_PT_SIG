// Package attendance_sync handles synchronization of attendance data from Fingerspot devices
// to the SIG attendance system. This service coordinates between the device integration
// layer and the attendance processing logic.
package attendance_sync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/attendance"
	"github.com/suryaintigas/absensi-backend/internal/employee"
	"github.com/suryaintigas/absensi-backend/internal/fingerspot"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

// Service handles attendance synchronization from Fingerspot devices
type Service struct {
	fingerspotService *fingerspot.Service
	employeeService   *employee.Service
	attendanceService *attendance.Service
}

// NewService creates a new attendance sync service
func NewService(
	fingerspotService *fingerspot.Service,
	employeeService *employee.Service,
	attendanceService *attendance.Service,
) *Service {
	return &Service{
		fingerspotService: fingerspotService,
		employeeService:   employeeService,
		attendanceService: attendanceService,
	}
}

// SyncResult represents the result of an attendance sync operation
type SyncResult struct {
	Success      bool
	Processed    int
	Failed       int
	Skipped      int
	Duplicate    int
	Total        int
	StartedAt    time.Time
	CompletedAt  time.Time
	ErrorMessage string
	Details      []SyncDetail
}

// SyncDetail provides detailed information about individual attendance sync operations
type SyncDetail struct {
	ExternalID     string
	DeviceUserID   string
	EmployeeID     uuid.UUID
	AttendanceType string
	AttendanceTime time.Time
	Status         string // SUCCESS, FAILED, SKIPPED, DUPLICATE
	Message        string
}

// SyncFromDevice synchronizes attendance logs from a Fingerspot device
func (s *Service) SyncFromDevice(ctx context.Context, deviceID uuid.UUID, from, to *time.Time) (*SyncResult, error) {
	startTime := time.Now()
	slog.Info("attendance_sync: starting sync from device",
		"device_id", deviceID,
		"from", from,
		"to", to)

	result := &SyncResult{
		StartedAt: startTime,
		Details:   make([]SyncDetail, 0),
	}

	// Sync attendance from device
	syncResult, err := s.fingerspotService.SyncAttendanceFromDevice(ctx)
	if err != nil {
		slog.Error("attendance_sync: failed to sync from device",
			"device_id", deviceID,
			"error", err.Error())
		result.Success = false
		result.ErrorMessage = err.Error()
		result.CompletedAt = time.Now()
		return result, fmt.Errorf("failed to sync attendance from device: %w", err)
	}

	// Process sync operation details
	// Note: The actual attendance logs would need to be processed separately
	// This is a simplified implementation that tracks sync operation status
	for _, log := range syncResult.Details {
		detail := SyncDetail{
			ExternalID: log.ExternalID,
			Status:     log.Status,
			Message:    log.Message,
		}

		switch log.Status {
		case "SUCCESS":
			result.Processed++
		case "FAILED":
			result.Failed++
		case "SKIPPED":
			result.Skipped++
		}

		result.Details = append(result.Details, detail)
	}

	result.Total = syncResult.Total
	result.Success = syncResult.Success
	result.CompletedAt = time.Now()

	slog.Info("attendance_sync: sync completed",
		"device_id", deviceID,
		"total", result.Total,
		"processed", result.Processed,
		"failed", result.Failed,
		"skipped", result.Skipped,
		"duration_ms", result.CompletedAt.Sub(startTime).Milliseconds())

	return result, nil
}

// SyncUsersToDevice synchronizes employee data to a Fingerspot device
func (s *Service) SyncUsersToDevice(ctx context.Context, deviceID uuid.UUID) (*SyncResult, error) {
	startTime := time.Now()
	slog.Info("attendance_sync: starting user sync to device", "device_id", deviceID)

	result := &SyncResult{
		StartedAt: startTime,
		Details:   make([]SyncDetail, 0),
	}

	// Get all active employees
	employees, _, err := s.employeeService.List(ctx, employee.Filter{
		Status: employee.StatusActive,
	}, pagination.Params{
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		slog.Error("attendance_sync: failed to get employees",
			"device_id", deviceID,
			"error", err.Error())
		result.Success = false
		result.ErrorMessage = err.Error()
		result.CompletedAt = time.Now()
		return result, fmt.Errorf("failed to get employees: %w", err)
	}

	// Convert employees to device users
	deviceUsers := make([]fingerspot.DeviceUser, 0, len(employees))
	for _, emp := range employees {
		deviceUsers = append(deviceUsers, fingerspot.DeviceUser{
			UserID:      emp.DeviceUserID,
			Name:        emp.Name,
			EmployeeID:  emp.ID,
			UserData:    "",
			Enabled:     emp.Status == employee.StatusActive,
			Fingerprint: emp.BiometricID != "", // Simplified logic
			Face:        emp.BiometricID != "", // Simplified logic
		})
	}

	// Sync users to device
	syncResult, err := s.fingerspotService.SyncUsersToDevice(ctx, deviceUsers)
	if err != nil {
		slog.Error("attendance_sync: failed to sync users to device",
			"device_id", deviceID,
			"error", err.Error())
		result.Success = false
		result.ErrorMessage = err.Error()
		result.CompletedAt = time.Now()
		return result, fmt.Errorf("failed to sync users to device: %w", err)
	}

	// Convert sync details to our format
	for _, detail := range syncResult.Details {
		result.Details = append(result.Details, SyncDetail{
			ExternalID: detail.ExternalID,
			Status:     detail.Status,
			Message:    detail.Message,
		})
	}

	result.Total = syncResult.Total
	result.Processed = syncResult.Processed
	result.Failed = syncResult.Failed
	result.Success = syncResult.Success
	result.CompletedAt = time.Now()

	slog.Info("attendance_sync: user sync completed",
		"device_id", deviceID,
		"total", result.Total,
		"processed", result.Processed,
		"failed", result.Failed,
		"duration_ms", result.CompletedAt.Sub(startTime).Milliseconds())

	return result, nil
}
