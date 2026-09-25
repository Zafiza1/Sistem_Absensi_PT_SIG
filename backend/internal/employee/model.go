// Package employee manages the people whose attendance is tracked —
// distinct from internal/auth's User, which is a dashboard login account.
package employee

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"
)

// Employee is the core master-data record every attendance check-in/out
// (Phase 4) and face profile (Phase 5) attaches to.
// Updated for Fingerspot device integration with device user ID mapping.
type Employee struct {
	ID             uuid.UUID
	EmployeeNumber string
	Name           string
	Email          *string
	Phone          *string
	DepartmentID   *uuid.UUID
	PositionID     *uuid.UUID
	ShiftID        *uuid.UUID
	Status         string
	// BaseSalary is "Gaji Pokok" in whole Rupiah, used by internal/payroll
	// to compute late-arrival deductions.
	BaseSalary int64
	// DeviceUserID is the user ID on the biometric device (e.g., Fingerspot user ID)
	DeviceUserID string
	// BiometricID is the biometric template ID on the device (fingerprint/face ID)
	BiometricID string
	// DeviceUserData contains additional device-specific user data
	DeviceUserData map[string]interface{}
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time

	// Denormalized display names, populated by List/FindByID via a JOIN so
	// the dashboard doesn't need three extra round-trips per row. Empty
	// when the corresponding *ID is nil.
	DepartmentName string
	PositionName   string
	ShiftName      string
}
