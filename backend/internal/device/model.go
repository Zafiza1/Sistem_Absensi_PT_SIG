// Package device manages the office tablets registered to run the
// attendance app. A tablet not registered here must never be allowed to
// submit attendance (enforced from Phase 4 onward, once check-in/out exists).
package device

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"

	// OnlineThreshold: a device is considered "online" if it has checked in
	// (LastSeenAt) within this window. This is derived at read time, never
	// stored, so it can't go stale the moment a tablet loses network
	// without ever calling back.
	OnlineThreshold = 5 * time.Minute
)

type Device struct {
	ID         uuid.UUID
	DeviceName string
	DeviceCode string
	Location   string
	Status     string
	AppVersion string
	LastSeenAt *time.Time
	LastSyncAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// IsOnline reports whether the device has been seen within OnlineThreshold
// of now.
func (d *Device) IsOnline(now time.Time) bool {
	return d.LastSeenAt != nil && now.Sub(*d.LastSeenAt) <= OnlineThreshold
}
