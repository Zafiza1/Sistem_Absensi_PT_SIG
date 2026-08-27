package attendance

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/device"
	"github.com/suryaintigas/absensi-backend/internal/employee"
	"github.com/suryaintigas/absensi-backend/internal/schedule"
	"github.com/suryaintigas/absensi-backend/internal/shift"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

// Caller-facing errors. The backend never trusts the tablet — every one of
// these is a re-validation of something the client claimed, per the spec's
// explicit requirement that check-in/out re-check employee, device,
// schedule, and shift server-side.
var (
	ErrEmployeeNotFound    = errors.New("attendance: employee not found or inactive")
	ErrDeviceNotRegistered = errors.New("attendance: device not registered or inactive")
	ErrNoShiftAssigned     = errors.New("attendance: employee has no shift assigned")
	ErrAlreadyCheckedIn    = errors.New("attendance: employee already has an attendance record today")
	ErrNoOpenCheckIn       = errors.New("attendance: no active check-in found for this employee")
	// ErrAlreadyCheckedOut is defined in repository.go and reused here
	// unchanged — CompleteCheckOut returns it directly when its guarded
	// UPDATE affects zero rows.
)

// jakarta is the fixed timezone every date/day-of-week/lateness
// calculation is performed in, regardless of the server's own timezone —
// PT Surya Inti Gas operates in one timezone, so hardcoding it keeps this
// logic simple and correct rather than plumbing a timezone through every
// call. Falls back to a fixed UTC+7 offset if the container image is ever
// missing tzdata (ours isn't — see backend/Dockerfile).
var jakarta = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}()

// Service implements check-in/check-out and attendance history. It depends
// on the other modules' Repository interfaces (not their Services) to
// re-validate employee/device/shift/schedule directly against the
// database, independent of any business rules those modules' own services
// might add later.
type Service struct {
	repo         Repository
	employeeRepo employee.Repository
	deviceRepo   device.Repository
	shiftRepo    shift.Repository
	scheduleRepo schedule.Repository
}

func NewService(
	repo Repository,
	employeeRepo employee.Repository,
	deviceRepo device.Repository,
	shiftRepo shift.Repository,
	scheduleRepo schedule.Repository,
) *Service {
	return &Service{
		repo:         repo,
		employeeRepo: employeeRepo,
		deviceRepo:   deviceRepo,
		shiftRepo:    shiftRepo,
		scheduleRepo: scheduleRepo,
	}
}

// CheckIn validates the employee and device, resolves the shift in effect
// for today, computes on-time/late status, and records the attendance. The
// check-in timestamp is always the server's clock — a tablet's clock is
// not trusted for something that determines whether pay is docked.
func (s *Service) CheckIn(ctx context.Context, employeeID uuid.UUID, deviceCode string) (*Attendance, error) {
	emp, err := s.validEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	dev, err := s.validDevice(ctx, deviceCode)
	if err != nil {
		return nil, err
	}

	now := time.Now().In(jakarta)
	today := dateOnly(now)

	sh, err := s.resolveShift(ctx, emp, isoWeekday(now))
	if err != nil {
		return nil, err
	}

	status, lateMinutes := computeCheckInStatus(now, sh)

	att := &Attendance{
		EmployeeID:      emp.ID,
		ShiftID:         &sh.ID,
		AttendanceDate:  today,
		CheckInAt:       &now,
		CheckInDeviceID: &dev.ID,
		Status:          status,
		LateMinutes:     lateMinutes,
	}
	if err := s.repo.Create(ctx, att); err != nil {
		if errors.Is(err, ErrDuplicateCheckIn) {
			return nil, ErrAlreadyCheckedIn
		}
		return nil, err
	}

	s.touchDevice(ctx, dev.ID)
	return att, nil
}

// CheckOut validates the employee and device, finds the employee's open
// (not yet checked out) attendance record — regardless of its
// attendance_date, so an overnight shift crossing midnight still resolves
// to the right record — and completes it with a working-duration
// calculation.
func (s *Service) CheckOut(ctx context.Context, employeeID uuid.UUID, deviceCode string) (*Attendance, error) {
	emp, err := s.validEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	dev, err := s.validDevice(ctx, deviceCode)
	if err != nil {
		return nil, err
	}

	att, err := s.repo.FindOpenByEmployee(ctx, emp.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNoOpenCheckIn
		}
		return nil, err
	}

	now := time.Now().In(jakarta)
	workingMinutes := int(now.Sub(*att.CheckInAt).Minutes())
	if workingMinutes < 0 {
		workingMinutes = 0
	}

	// CompleteCheckOut already returns the exported ErrAlreadyCheckedOut
	// verbatim on a race (e.g. two check-out calls in flight), so it can be
	// passed straight through to the handler.
	if err := s.repo.CompleteCheckOut(ctx, att.ID, now, dev.ID, workingMinutes); err != nil {
		return nil, err
	}

	s.touchDevice(ctx, dev.ID)
	return s.repo.FindByID(ctx, att.ID)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Attendance, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, f Filter, p pagination.Params) ([]Attendance, int64, error) {
	return s.repo.List(ctx, f, p)
}

func (s *Service) validEmployee(ctx context.Context, id uuid.UUID) (*employee.Employee, error) {
	emp, err := s.employeeRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrEmployeeNotFound
	}
	if emp.Status != employee.StatusActive {
		return nil, ErrEmployeeNotFound
	}
	return emp, nil
}

func (s *Service) validDevice(ctx context.Context, code string) (*device.Device, error) {
	dev, err := s.deviceRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, ErrDeviceNotRegistered
	}
	if dev.Status != device.StatusActive {
		return nil, ErrDeviceNotRegistered
	}
	return dev, nil
}

// resolveShift prefers a per-weekday schedule override, falling back to
// the employee's default shift.
func (s *Service) resolveShift(ctx context.Context, emp *employee.Employee, dayOfWeek int) (*shift.Shift, error) {
	sc, err := s.scheduleRepo.FindForEmployeeAndDay(ctx, emp.ID, dayOfWeek)
	switch {
	case err == nil:
		return s.shiftRepo.FindByID(ctx, sc.ShiftID)
	case errors.Is(err, schedule.ErrNotFound):
		if emp.ShiftID == nil {
			return nil, ErrNoShiftAssigned
		}
		return s.shiftRepo.FindByID(ctx, *emp.ShiftID)
	default:
		return nil, err
	}
}

// touchDevice best-effort records that the device was just seen. A failure
// here must never fail the attendance request that triggered it — it only
// feeds the dashboard's online/offline indicator.
func (s *Service) touchDevice(ctx context.Context, deviceID uuid.UUID) {
	if err := s.deviceRepo.Touch(ctx, deviceID, time.Now()); err != nil {
		slog.Warn("attendance_device_touch_failed", slog.String("error", err.Error()), slog.String("device_id", deviceID.String()))
	}
}

// computeCheckInStatus compares now against sh's start time on now's
// calendar date. A check-in before the shift start, or within
// LateToleranceMinutes after it, is on-time; otherwise late, with
// lateMinutes counting every minute past the official start time (not just
// past the tolerance) — "how late were they", matching the spec's "Late
// Duration" field.
func computeCheckInStatus(now time.Time, sh *shift.Shift) (status string, lateMinutes int) {
	startClock, err := time.Parse("15:04", sh.StartTime)
	if err != nil {
		// Shift rows are validated at write time (internal/shift), so this
		// should be unreachable; treat as on-time rather than panic.
		return StatusOnTime, 0
	}
	shiftStart := time.Date(now.Year(), now.Month(), now.Day(), startClock.Hour(), startClock.Minute(), 0, 0, now.Location())

	diff := now.Sub(shiftStart)
	tolerance := time.Duration(sh.LateToleranceMinutes) * time.Minute
	if diff <= tolerance {
		return StatusOnTime, 0
	}
	return StatusLate, int(diff.Minutes())
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// isoWeekday converts Go's time.Weekday (Sunday=0..Saturday=6) to the ISO
// 8601 convention used by work_schedules.day_of_week (Monday=1..Sunday=7).
func isoWeekday(t time.Time) int {
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}
