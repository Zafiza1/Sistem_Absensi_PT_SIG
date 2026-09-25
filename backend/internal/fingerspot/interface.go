// Package fingerspot provides abstraction layer for Fingerspot biometric device integration.
// This interface-based approach allows the system to be device-agnostic and enables
// easy switching between different device implementations or vendors in the future.
package fingerspot

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// DeviceIntegration defines the contract for biometric device integration.
// This interface abstracts the specific details of communicating with Fingerspot
// devices, allowing the business logic to remain independent of the vendor implementation.
type DeviceIntegration interface {
	// Connect establishes connection to the device
	Connect(ctx context.Context) error

	// Disconnect closes the device connection
	Disconnect(ctx context.Context) error

	// Authenticate performs authentication with the device using credentials
	Authenticate(ctx context.Context) error

	// GetDeviceInfo retrieves basic information about the device
	GetDeviceInfo(ctx context.Context) (*DeviceInfo, error)

	// GetDeviceStatus retrieves current operational status of the device
	GetDeviceStatus(ctx context.Context) (*DeviceStatus, error)

	// GetUsers retrieves list of users enrolled on the device
	GetUsers(ctx context.Context) ([]DeviceUser, error)

	// SyncUsers synchronizes users from SIG system to the device
	SyncUsers(ctx context.Context, users []DeviceUser) error

	// GetAttendanceLogs retrieves attendance logs from the device
	GetAttendanceLogs(ctx context.Context, from, to time.Time) ([]AttendanceLog, error)

	// SyncAttendance synchronizes attendance logs from device to SIG system
	SyncAttendance(ctx context.Context) ([]AttendanceLog, error)

	// IsConnected checks if the device is currently connected
	IsConnected() bool
}

// DeviceInfo contains basic information about the biometric device
type DeviceInfo struct {
	DeviceID     string
	DeviceName   string
	SerialNumber string
	DeviceType   string
	Firmware     string
	IPAddress    string
	Port         int
}

// DeviceStatus represents the current operational status of the device
type DeviceStatus struct {
	IsOnline      bool
	LastSeenAt   time.Time
	StorageUsed  int
	StorageTotal int
	UserCount    int
	LogCount     int
	BatteryLevel int // 0-100, or -1 if not applicable
}

// DeviceUser represents a user enrolled on the biometric device
type DeviceUser struct {
	UserID      string
	Name        string
	EmployeeID  uuid.UUID // Reference to SIG employee
	UserData    string    // Additional device-specific data
	Enabled     bool
	Fingerprint bool      // Has fingerprint enrolled
	Face        bool      // Has face enrolled
}

// AttendanceLog represents a raw attendance record from the device
type AttendanceLog struct {
	ExternalID    string    // Device-specific ID for idempotency
	DeviceUserID  string    // User ID on the device
	EmployeeID    uuid.UUID // Reference to SIG employee (may be empty initially)
	AttendanceType string   // "CHECK_IN" or "CHECK_OUT"
	AttendanceTime time.Time
	DeviceID      uuid.UUID // Reference to SIG device record
	DeviceSN      string    // Device serial number
	VerifyMode    string    // How the user was verified (fingerprint, face, password, etc.)
	WorkCode      string    // Optional work code
	RawData       string    // Raw data from device for traceability
}

// SyncResult represents the result of a synchronization operation
type SyncResult struct {
	Success       bool
	Processed     int
	Failed        int
	Total         int
	StartedAt     time.Time
	CompletedAt   time.Time
	ErrorMessage  string
	Details       []SyncDetail
}

// SyncDetail provides detailed information about individual sync operations
type SyncDetail struct {
	ExternalID string
	Status     string // "SUCCESS", "FAILED", "SKIPPED"
	Message    string
	Timestamp  time.Time
}
