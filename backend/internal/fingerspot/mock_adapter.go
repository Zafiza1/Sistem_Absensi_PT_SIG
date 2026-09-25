// Package fingerspot provides a mock adapter for development and testing.
// This is a temporary implementation until the official Fingerspot SDK/API
// documentation is available. It simulates device behavior for development purposes.
//
// IMPORTANT: This mock adapter should only be used for development and testing.
// Replace with the real Fingerspot adapter when official SDK/API documentation is available.
package fingerspot

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MockAdapter is a mock implementation of DeviceIntegration for development/testing.
// It simulates Fingerspot device behavior without requiring actual hardware.
type MockAdapter struct {
	config       Config
	connected    bool
	mu           sync.RWMutex
	mockUsers    []DeviceUser
	mockLogs     []AttendanceLog
	lastSync     time.Time
}

// Config holds configuration for the mock adapter
type Config struct {
	DeviceID    string
	IPAddress   string
	Port        int
	APIKey      string
	APISecret   string
	EnableDelay bool // Simulate network latency
}

// NewMockAdapter creates a new mock Fingerspot adapter
func NewMockAdapter(config Config) *MockAdapter {
	return &MockAdapter{
		config:    config,
		connected: false,
		mockUsers: generateMockUsers(),
		mockLogs:  generateMockLogs(),
	}
}

// Connect simulates establishing connection to the device
func (m *MockAdapter) Connect(ctx context.Context) error {
	m.simulateDelay("connect")

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connected {
		return fmt.Errorf("already connected to device")
	}

	slog.Info("mock_adapter: connecting to device",
		"device_id", m.config.DeviceID,
		"ip", m.config.IPAddress,
		"port", m.config.Port)

	// Simulate connection delay
	time.Sleep(100 * time.Millisecond)

	m.connected = true
	slog.Info("mock_adapter: connected successfully")
	return nil
}

// Disconnect simulates closing the device connection
func (m *MockAdapter) Disconnect(ctx context.Context) error {
	m.simulateDelay("disconnect")

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return fmt.Errorf("not connected to device")
	}

	slog.Info("mock_adapter: disconnecting from device")
	m.connected = false
	slog.Info("mock_adapter: disconnected successfully")
	return nil
}

// Authenticate simulates authentication with the device
func (m *MockAdapter) Authenticate(ctx context.Context) error {
	m.simulateDelay("authenticate")

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return fmt.Errorf("not connected to device")
	}

	slog.Info("mock_adapter: authenticating with device")

	// Simulate authentication delay
	time.Sleep(50 * time.Millisecond)

	// Mock authentication always succeeds for development
	slog.Info("mock_adapter: authentication successful")
	return nil
}

// GetDeviceInfo retrieves mock device information
func (m *MockAdapter) GetDeviceInfo(ctx context.Context) (*DeviceInfo, error) {
	m.simulateDelay("get_device_info")

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return nil, fmt.Errorf("not connected to device")
	}

	return &DeviceInfo{
		DeviceID:     m.config.DeviceID,
		DeviceName:   "Fingerspot Mock Device",
		SerialNumber: "FS-MOCK-" + m.config.DeviceID,
		DeviceType:   "FINGERSPOT",
		Firmware:     "1.0.0-mock",
		IPAddress:    m.config.IPAddress,
		Port:         m.config.Port,
	}, nil
}

// GetDeviceStatus retrieves mock device status
func (m *MockAdapter) GetDeviceStatus(ctx context.Context) (*DeviceStatus, error) {
	m.simulateDelay("get_device_status")

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return nil, fmt.Errorf("not connected to device")
	}

	return &DeviceStatus{
		IsOnline:      true,
		LastSeenAt:   time.Now(),
		StorageUsed:  rand.Intn(1000),
		StorageTotal: 10000,
		UserCount:    len(m.mockUsers),
		LogCount:     len(m.mockLogs),
		BatteryLevel: -1, // Not applicable for desktop devices
	}, nil
}

// GetUsers retrieves mock users from the device
func (m *MockAdapter) GetUsers(ctx context.Context) ([]DeviceUser, error) {
	m.simulateDelay("get_users")

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return nil, fmt.Errorf("not connected to device")
	}

	slog.Info("mock_adapter: retrieved users", "count", len(m.mockUsers))
	return m.mockUsers, nil
}

// SyncUsers simulates synchronizing users to the device
func (m *MockAdapter) SyncUsers(ctx context.Context, users []DeviceUser) error {
	m.simulateDelay("sync_users")

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return fmt.Errorf("not connected to device")
	}

	slog.Info("mock_adapter: syncing users", "count", len(users))

	// Simulate sync delay
	time.Sleep(200 * time.Millisecond)

	// Update mock users
	m.mockUsers = users
	m.lastSync = time.Now()

	slog.Info("mock_adapter: users synced successfully")
	return nil
}

// GetAttendanceLogs retrieves mock attendance logs from the device
func (m *MockAdapter) GetAttendanceLogs(ctx context.Context, from, to time.Time) ([]AttendanceLog, error) {
	m.simulateDelay("get_attendance_logs")

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return nil, fmt.Errorf("not connected to device")
	}

	// Filter logs by date range
	var filteredLogs []AttendanceLog
	for _, log := range m.mockLogs {
		if (from.IsZero() || log.AttendanceTime.After(from) || log.AttendanceTime.Equal(from)) &&
			(to.IsZero() || log.AttendanceTime.Before(to) || log.AttendanceTime.Equal(to)) {
			filteredLogs = append(filteredLogs, log)
		}
	}

	slog.Info("mock_adapter: retrieved attendance logs",
		"count", len(filteredLogs),
		"from", from,
		"to", to)

	return filteredLogs, nil
}

// SyncAttendance simulates synchronizing attendance logs from the device
func (m *MockAdapter) SyncAttendance(ctx context.Context) ([]AttendanceLog, error) {
	m.simulateDelay("sync_attendance")

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return nil, fmt.Errorf("not connected to device")
	}

	slog.Info("mock_adapter: syncing attendance logs")

	// Simulate sync delay
	time.Sleep(300 * time.Millisecond)

	// In a real implementation, this would fetch new logs since last sync
	// For mock, we return all logs and update last sync time
	m.lastSync = time.Now()

	slog.Info("mock_adapter: attendance synced successfully",
		"count", len(m.mockLogs))

	return m.mockLogs, nil
}

// IsConnected checks if the mock device is connected
func (m *MockAdapter) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// simulateDelay adds optional delay to simulate network latency
func (m *MockAdapter) simulateDelay(operation string) {
	if m.config.EnableDelay {
		delay := time.Duration(rand.Intn(100)) * time.Millisecond
		slog.Debug("mock_adapter: simulating delay",
			"operation", operation,
			"delay_ms", delay.Milliseconds())
		time.Sleep(delay)
	}
}

// generateMockUsers creates mock device users for testing
func generateMockUsers() []DeviceUser {
	return []DeviceUser{
		{
			UserID:     "USER001",
			Name:       "John Doe",
			EmployeeID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			UserData:   "",
			Enabled:    true,
			Fingerprint: true,
			Face:       false,
		},
		{
			UserID:     "USER002",
			Name:       "Jane Smith",
			EmployeeID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			UserData:   "",
			Enabled:    true,
			Fingerprint: true,
			Face:       true,
		},
		{
			UserID:     "USER003",
			Name:       "Bob Johnson",
			EmployeeID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			UserData:   "",
			Enabled:    true,
			Fingerprint: false,
			Face:       true,
		},
	}
}

// generateMockLogs creates mock attendance logs for testing
func generateMockLogs() []AttendanceLog {
	now := time.Now()
	deviceID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	return []AttendanceLog{
		{
			ExternalID:     "LOG001",
			DeviceUserID:   "USER001",
			EmployeeID:     uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			AttendanceType: "CHECK_IN",
			AttendanceTime: now.Add(-2 * time.Hour),
			DeviceID:       deviceID,
			DeviceSN:       "FS-MOCK-001",
			VerifyMode:     "Fingerprint",
			WorkCode:       "",
			RawData:        "",
		},
		{
			ExternalID:     "LOG002",
			DeviceUserID:   "USER002",
			EmployeeID:     uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			AttendanceType: "CHECK_IN",
			AttendanceTime: now.Add(-1 * time.Hour),
			DeviceID:       deviceID,
			DeviceSN:       "FS-MOCK-001",
			VerifyMode:     "Face",
			WorkCode:       "",
			RawData:        "",
		},
		{
			ExternalID:     "LOG003",
			DeviceUserID:   "USER001",
			EmployeeID:     uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			AttendanceType: "CHECK_OUT",
			AttendanceTime: now.Add(-30 * time.Minute),
			DeviceID:       deviceID,
			DeviceSN:       "FS-MOCK-001",
			VerifyMode:     "Fingerprint",
			WorkCode:       "",
			RawData:        "",
		},
	}
}
