// Package device manages biometric devices (Fingerspot) registered for attendance.
// Devices not registered here must never be allowed to sync attendance data.
// This module has been updated from tablet-oriented to Fingerspot device-oriented.
package device

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"

	// DeviceType constants
	DeviceTypeFingerspot = "FINGERSPOT"
	DeviceTypeTablet     = "TABLET"
	DeviceTypeOther      = "OTHER"

	// ConnectionStatus constants
	ConnectionStatusConnected    = "CONNECTED"
	ConnectionStatusDisconnected = "DISCONNECTED"
	ConnectionStatusError        = "ERROR"
	ConnectionStatusSyncing      = "SYNCING"

	// SyncStatus constants
	SyncStatusIdle    = "IDLE"
	SyncStatusSyncing = "SYNCING"
	SyncStatusSuccess = "SUCCESS"
	SyncStatusFailed  = "FAILED"

	// OnlineThreshold: a device is considered "online" if it has checked in
	// (LastSeenAt) within this window. This is derived at read time, never
	// stored, so it can't go stale the moment a device loses network
	// without ever calling back.
	OnlineThreshold = 5 * time.Minute
)

type Device struct {
	ID               uuid.UUID
	DeviceName       string
	DeviceCode       string
	Location         string
	Status           string
	DeviceType       string
	SerialNumber     string
	IPAddress        string
	Port             int
	ConnectionStatus string
	AppVersion       string
	LastSeenAt       *time.Time
	LastSyncAt       *time.Time
	SyncStatus       string
	ErrorMessage     string
	DeviceConfig     map[string]interface{}
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// IsOnline reports whether the device has been seen within OnlineThreshold
// of now.
func (d *Device) IsOnline(now time.Time) bool {
	return d.LastSeenAt != nil && now.Sub(*d.LastSeenAt) <= OnlineThreshold
}

// IsConnected reports whether the device connection status is CONNECTED
func (d *Device) IsConnected() bool {
	return d.ConnectionStatus == ConnectionStatusConnected
}

// IsSyncing reports whether the device is currently syncing
func (d *Device) IsSyncing() bool {
	return d.SyncStatus == SyncStatusSyncing
}
