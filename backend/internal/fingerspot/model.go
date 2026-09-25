// Package fingerspot provides data models for Fingerspot device integration.
// These models are used for database operations and API responses.
package fingerspot

import (
	"time"

	"github.com/google/uuid"
)

// Device represents a Fingerspot device in the SIG system
type Device struct {
	ID               uuid.UUID
	DeviceName       string
	DeviceCode       string
	Location         string
	Status           string // ACTIVE, INACTIVE
	DeviceType       string // FINGERSPOT, TABLET, OTHER
	SerialNumber     string
	IPAddress        string
	Port             int
	ConnectionStatus string // CONNECTED, DISCONNECTED, ERROR, SYNCING
	AppVersion       string
	LastSeenAt       *time.Time
	LastSyncAt       *time.Time
	SyncStatus       string // IDLE, SYNCING, SUCCESS, FAILED
	ErrorMessage     string
	DeviceConfig     map[string]interface{}
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// DeviceUserMapping represents the mapping between SIG employees and device users
type DeviceUserMapping struct {
	ID              uuid.UUID
	EmployeeID      uuid.UUID
	DeviceID        uuid.UUID
	DeviceUserID    string
	BiometricID     string
	UserData        map[string]interface{}
	Enabled         bool
	FingerprintEnrolled bool
	FaceEnrolled    bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AttendanceLogRecord represents a raw attendance log from the device
type AttendanceLogRecord struct {
	ID              uuid.UUID
	EmployeeID      uuid.UUID
	DeviceID        uuid.UUID
	DeviceUserID    string
	AttendanceType  string // CHECK_IN, CHECK_OUT
	AttendanceTime  time.Time
	AttendanceDate  time.Time
	Source          string // FINGERSPOT, TABLET, MANUAL, API
	RawData         map[string]interface{}
	ProcessedAt     *time.Time
	ExternalID      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// SyncStatusRecord represents the status of a synchronization operation
type SyncStatusRecord struct {
	ID                uuid.UUID
	DeviceID          uuid.UUID
	SyncType          string // ATTENDANCE, USERS, CONFIG, FULL
	Status            string // PENDING, IN_PROGRESS, SUCCESS, FAILED, PARTIAL
	StartedAt         time.Time
	CompletedAt       *time.Time
	RecordsProcessed  int
	RecordsFailed     int
	RecordsTotal      int
	ErrorMessage      string
	Metadata          map[string]interface{}
	CreatedAt         time.Time
}

// AttendanceSyncRequest represents a request to sync attendance from a device
type AttendanceSyncRequest struct {
	DeviceID uuid.UUID
	FromDate *time.Time
	ToDate   *time.Time
}

// AttendanceSyncResponse represents the response from an attendance sync operation
type AttendanceSyncResponse struct {
	Success      bool
	Processed    int
	Failed       int
	Skipped      int
	Total        int
	StartedAt    time.Time
	CompletedAt  time.Time
	ErrorMessage string
	Details      []AttendanceSyncDetail
}

// AttendanceSyncDetail provides detailed information about individual attendance sync operations
type AttendanceSyncDetail struct {
	ExternalID     string
	EmployeeID     uuid.UUID
	AttendanceType string
	AttendanceTime time.Time
	Status         string // SUCCESS, FAILED, SKIPPED
	Message        string
}

// UserSyncRequest represents a request to sync users to a device
type UserSyncRequest struct {
	DeviceID uuid.UUID
	Users    []UserSyncItem
}

// UserSyncItem represents a single user to sync to the device
type UserSyncItem struct {
	EmployeeID          uuid.UUID
	DeviceUserID        string
	Name                string
	UserData            map[string]interface{}
	Enabled             bool
	FingerprintEnrolled bool
	FaceEnrolled        bool
}

// UserSyncResponse represents the response from a user sync operation
type UserSyncResponse struct {
	Success      bool
	Processed    int
	Failed       int
	Total        int
	StartedAt    time.Time
	CompletedAt  time.Time
	ErrorMessage string
	Details      []UserSyncDetail
}

// UserSyncDetail provides detailed information about individual user sync operations
type UserSyncDetail struct {
	EmployeeID uuid.UUID
	DeviceUserID string
	Status     string // SUCCESS, FAILED, SKIPPED
	Message    string
}

// DeviceConnectionRequest represents a request to connect to a device
type DeviceConnectionRequest struct {
	DeviceID uuid.UUID
}

// DeviceConnectionResponse represents the response from a device connection operation
type DeviceConnectionResponse struct {
	Success     bool
	DeviceInfo  DeviceInfo
	DeviceStatus DeviceStatus
	ConnectedAt time.Time
	Message     string
}

// DeviceDisconnectionRequest represents a request to disconnect from a device
type DeviceDisconnectionRequest struct {
	DeviceID uuid.UUID
}

// DeviceDisconnectionResponse represents the response from a device disconnection operation
type DeviceDisconnectionResponse struct {
	Success       bool
	DisconnectedAt time.Time
	Message       string
}
