// Package attendance implements check-in/check-out, late/working-duration
// calculation, and attendance history.
//
// The trust boundary here is deliberately different from every other
// module: check-in/check-out are called by the office tablet, which has no
// human dashboard login, so they are not protected by the JWT
// Authorization header the rest of the API requires. Instead, per the
// spec, the tablet's registered device_code is the credential — an
// unregistered or deactivated device cannot submit attendance. This is an
// interim posture appropriate for tablets on a private network; Phase 8
// deployment notes call out that it should sit behind additional network
// controls (VPN/firewall to the office network) before being exposed
// publicly, and Phase 5 may add a proper per-device credential once the
// Flutter app's actual auth needs are known.
package attendance

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusOnTime     = "ON_TIME"
	StatusLate       = "LATE"
	StatusCheckedOut = "CHECKED_OUT"
	// StatusAbsent and StatusIncomplete are never written by this package —
	// see the migration's comment. They exist here only so callers that
	// derive a display status (Phase 6 reporting) share the same constants.
	StatusAbsent     = "ABSENT"
	StatusIncomplete = "INCOMPLETE"
)

// Attendance is one employee's attendance record for one calendar day.
type Attendance struct {
	ID                     uuid.UUID
	EmployeeID             uuid.UUID
	ShiftID                *uuid.UUID
	AttendanceDate         time.Time
	CheckInAt              *time.Time
	CheckInDeviceID        *uuid.UUID
	CheckOutAt             *time.Time
	CheckOutDeviceID       *uuid.UUID
	Status                 string
	LateMinutes            int
	WorkingDurationMinutes *int
	CreatedAt              time.Time
	UpdatedAt              time.Time

	// Denormalized display fields populated by JOINs in the repository.
	EmployeeName       string
	EmployeeNumber     string
	ShiftName          string
	CheckInDeviceName  string
	CheckOutDeviceName string
}
