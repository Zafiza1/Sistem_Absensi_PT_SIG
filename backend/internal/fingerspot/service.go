// Package fingerspot provides the service layer for Fingerspot device integration.
// This service coordinates between the device adapter and the business logic,
// handling synchronization, error handling, and logging.
package fingerspot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Service handles the business logic for Fingerspot device integration
type Service struct {
	adapter DeviceIntegration
	deviceID uuid.UUID
}

// NewService creates a new Fingerspot integration service
func NewService(adapter DeviceIntegration, deviceID uuid.UUID) *Service {
	return &Service{
		adapter:  adapter,
		deviceID: deviceID,
	}
}

// ConnectToDevice establishes connection to the Fingerspot device
func (s *Service) ConnectToDevice(ctx context.Context) error {
	slog.Info("fingerspot_service: connecting to device", "device_id", s.deviceID)

	if err := s.adapter.Connect(ctx); err != nil {
		slog.Error("fingerspot_service: connection failed",
			"device_id", s.deviceID,
			"error", err.Error())
		return fmt.Errorf("failed to connect to device: %w", err)
	}

	if err := s.adapter.Authenticate(ctx); err != nil {
		slog.Error("fingerspot_service: authentication failed",
			"device_id", s.deviceID,
			"error", err.Error())
		s.adapter.Disconnect(ctx) // Clean up connection
		return fmt.Errorf("failed to authenticate with device: %w", err)
	}

	slog.Info("fingerspot_service: device connected and authenticated",
		"device_id", s.deviceID)
	return nil
}

// DisconnectFromDevice closes the connection to the Fingerspot device
func (s *Service) DisconnectFromDevice(ctx context.Context) error {
	slog.Info("fingerspot_service: disconnecting from device", "device_id", s.deviceID)

	if err := s.adapter.Disconnect(ctx); err != nil {
		slog.Error("fingerspot_service: disconnection failed",
			"device_id", s.deviceID,
			"error", err.Error())
		return fmt.Errorf("failed to disconnect from device: %w", err)
	}

	slog.Info("fingerspot_service: device disconnected", "device_id", s.deviceID)
	return nil
}

// GetDeviceStatus retrieves the current status of the device
func (s *Service) GetDeviceStatus(ctx context.Context) (*DeviceStatus, error) {
	slog.Debug("fingerspot_service: getting device status", "device_id", s.deviceID)

	status, err := s.adapter.GetDeviceStatus(ctx)
	if err != nil {
		slog.Error("fingerspot_service: failed to get device status",
			"device_id", s.deviceID,
			"error", err.Error())
		return nil, fmt.Errorf("failed to get device status: %w", err)
	}

	return status, nil
}

// GetDeviceInfo retrieves information about the device
func (s *Service) GetDeviceInfo(ctx context.Context) (*DeviceInfo, error) {
	slog.Debug("fingerspot_service: getting device info", "device_id", s.deviceID)

	info, err := s.adapter.GetDeviceInfo(ctx)
	if err != nil {
		slog.Error("fingerspot_service: failed to get device info",
			"device_id", s.deviceID,
			"error", err.Error())
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	return info, nil
}

// SyncUsersToDevice synchronizes users from SIG system to the device
func (s *Service) SyncUsersToDevice(ctx context.Context, users []DeviceUser) (*SyncResult, error) {
	startTime := time.Now()
	slog.Info("fingerspot_service: starting user sync",
		"device_id", s.deviceID,
		"user_count", len(users))

	result := &SyncResult{
		StartedAt: startTime,
		Total:     len(users),
		Details:   make([]SyncDetail, 0, len(users)),
	}

	// Sync users to device
	if err := s.adapter.SyncUsers(ctx, users); err != nil {
		slog.Error("fingerspot_service: user sync failed",
			"device_id", s.deviceID,
			"error", err.Error())
		result.Success = false
		result.ErrorMessage = err.Error()
		result.CompletedAt = time.Now()
		return result, fmt.Errorf("failed to sync users: %w", err)
	}

	// Mark all as successful for mock implementation
	for _, user := range users {
		result.Details = append(result.Details, SyncDetail{
			ExternalID: user.UserID,
			Status:     "SUCCESS",
			Message:    "User synced successfully",
			Timestamp:  time.Now(),
		})
	}

	result.Success = true
	result.Processed = len(users)
	result.CompletedAt = time.Now()

	slog.Info("fingerspot_service: user sync completed",
		"device_id", s.deviceID,
		"processed", result.Processed,
		"duration_ms", result.CompletedAt.Sub(startTime).Milliseconds())

	return result, nil
}

// SyncAttendanceFromDevice synchronizes attendance logs from device to SIG system
func (s *Service) SyncAttendanceFromDevice(ctx context.Context) (*SyncResult, error) {
	startTime := time.Now()
	slog.Info("fingerspot_service: starting attendance sync", "device_id", s.deviceID)

	result := &SyncResult{
		StartedAt: startTime,
		Details:   make([]SyncDetail, 0),
	}

	// Get attendance logs from device
	logs, err := s.adapter.SyncAttendance(ctx)
	if err != nil {
		slog.Error("fingerspot_service: attendance sync failed",
			"device_id", s.deviceID,
			"error", err.Error())
		result.Success = false
		result.ErrorMessage = err.Error()
		result.CompletedAt = time.Now()
		return result, fmt.Errorf("failed to sync attendance: %w", err)
	}

	result.Total = len(logs)

	// Process each log
	for _, log := range logs {
		detail := SyncDetail{
			ExternalID: log.ExternalID,
			Timestamp:  time.Now(),
		}

		// In a real implementation, this would:
		// 1. Check if log already exists (idempotency)
		// 2. Map device user ID to employee ID
		// 3. Validate attendance data
		// 4. Store in attendance_logs table
		// 5. Process into attendances table

		// For now, mark as successful
		detail.Status = "SUCCESS"
		detail.Message = "Attendance log processed successfully"
		result.Details = append(result.Details, detail)
		result.Processed++
	}

	result.Success = true
	result.CompletedAt = time.Now()

	slog.Info("fingerspot_service: attendance sync completed",
		"device_id", s.deviceID,
		"processed", result.Processed,
		"duration_ms", result.CompletedAt.Sub(startTime).Milliseconds())

	return result, nil
}

// GetUsersFromDevice retrieves users enrolled on the device
func (s *Service) GetUsersFromDevice(ctx context.Context) ([]DeviceUser, error) {
	slog.Debug("fingerspot_service: getting users from device", "device_id", s.deviceID)

	users, err := s.adapter.GetUsers(ctx)
	if err != nil {
		slog.Error("fingerspot_service: failed to get users",
			"device_id", s.deviceID,
			"error", err.Error())
		return nil, fmt.Errorf("failed to get users from device: %w", err)
	}

	slog.Info("fingerspot_service: retrieved users",
		"device_id", s.deviceID,
		"count", len(users))

	return users, nil
}

// GetAttendanceLogsFromDevice retrieves attendance logs from the device
func (s *Service) GetAttendanceLogsFromDevice(ctx context.Context, from, to time.Time) ([]AttendanceLog, error) {
	slog.Debug("fingerspot_service: getting attendance logs from device",
		"device_id", s.deviceID,
		"from", from,
		"to", to)

	logs, err := s.adapter.GetAttendanceLogs(ctx, from, to)
	if err != nil {
		slog.Error("fingerspot_service: failed to get attendance logs",
			"device_id", s.deviceID,
			"error", err.Error())
		return nil, fmt.Errorf("failed to get attendance logs from device: %w", err)
	}

	slog.Info("fingerspot_service: retrieved attendance logs",
		"device_id", s.deviceID,
		"count", len(logs))

	return logs, nil
}

// IsDeviceConnected checks if the device is currently connected
func (s *Service) IsDeviceConnected() bool {
	return s.adapter.IsConnected()
}
