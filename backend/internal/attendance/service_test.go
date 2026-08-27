package attendance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/device"
	"github.com/suryaintigas/absensi-backend/internal/employee"
	"github.com/suryaintigas/absensi-backend/internal/schedule"
	"github.com/suryaintigas/absensi-backend/internal/shift"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
)

// --- computeCheckInStatus: pure-function coverage of the late/on-time math ---

func TestComputeCheckInStatus(t *testing.T) {
	sh := &shift.Shift{StartTime: "08:00", LateToleranceMinutes: 10}

	tests := []struct {
		name       string
		checkInHM  string // "HH:MM" on the same day as sh.StartTime
		wantStatus string
		wantLate   int
	}{
		{"well before shift start", "07:30", StatusOnTime, 0},
		{"exactly on time", "08:00", StatusOnTime, 0},
		{"within tolerance", "08:09", StatusOnTime, 0},
		{"exactly at tolerance boundary", "08:10", StatusOnTime, 0},
		{"one minute past tolerance", "08:11", StatusLate, 11},
		{"very late", "09:30", StatusLate, 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock, _ := time.Parse("15:04", tt.checkInHM)
			now := time.Date(2026, 1, 5, clock.Hour(), clock.Minute(), 0, 0, jakarta) // 2026-01-05 is a Monday

			status, late := computeCheckInStatus(now, sh)
			if status != tt.wantStatus {
				t.Errorf("status = %q, want %q", status, tt.wantStatus)
			}
			if late != tt.wantLate {
				t.Errorf("lateMinutes = %d, want %d", late, tt.wantLate)
			}
		})
	}
}

func TestIsoWeekday(t *testing.T) {
	tests := []struct {
		date string // YYYY-MM-DD
		want int
	}{
		{"2026-01-05", 1}, // Monday
		{"2026-01-11", 7}, // Sunday
		{"2026-01-08", 4}, // Thursday
	}
	for _, tt := range tests {
		d, _ := time.Parse("2006-01-02", tt.date)
		if got := isoWeekday(d); got != tt.want {
			t.Errorf("isoWeekday(%s) = %d, want %d", tt.date, got, tt.want)
		}
	}
}

// --- Service-level tests against fakes for every collaborator ---

type fakeEmployeeRepo struct {
	byID map[uuid.UUID]*employee.Employee
}

func (f *fakeEmployeeRepo) Create(context.Context, *employee.Employee) error { return nil }
func (f *fakeEmployeeRepo) FindByID(_ context.Context, id uuid.UUID) (*employee.Employee, error) {
	if e, ok := f.byID[id]; ok {
		return e, nil
	}
	return nil, employee.ErrNotFound
}
func (f *fakeEmployeeRepo) List(context.Context, employee.Filter, pagination.Params) ([]employee.Employee, int64, error) {
	return nil, 0, nil
}
func (f *fakeEmployeeRepo) Update(context.Context, *employee.Employee) error { return nil }
func (f *fakeEmployeeRepo) SoftDelete(context.Context, uuid.UUID) error      { return nil }

var _ employee.Repository = (*fakeEmployeeRepo)(nil)

type fakeDeviceRepo struct {
	byCode map[string]*device.Device
	seen   map[uuid.UUID]time.Time
}

func (f *fakeDeviceRepo) Create(context.Context, *device.Device) error { return nil }
func (f *fakeDeviceRepo) FindByID(_ context.Context, id uuid.UUID) (*device.Device, error) {
	for _, d := range f.byCode {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, device.ErrNotFound
}
func (f *fakeDeviceRepo) FindByCode(_ context.Context, code string) (*device.Device, error) {
	if d, ok := f.byCode[code]; ok {
		return d, nil
	}
	return nil, device.ErrNotFound
}
func (f *fakeDeviceRepo) List(context.Context, pagination.Params) ([]device.Device, int64, error) {
	return nil, 0, nil
}
func (f *fakeDeviceRepo) Update(context.Context, *device.Device) error { return nil }
func (f *fakeDeviceRepo) Delete(context.Context, uuid.UUID) error      { return nil }
func (f *fakeDeviceRepo) Touch(_ context.Context, id uuid.UUID, seenAt time.Time) error {
	if f.seen == nil {
		f.seen = map[uuid.UUID]time.Time{}
	}
	f.seen[id] = seenAt
	return nil
}

var _ device.Repository = (*fakeDeviceRepo)(nil)

type fakeShiftRepo struct{ byID map[uuid.UUID]*shift.Shift }

func (f *fakeShiftRepo) Create(context.Context, *shift.Shift) error { return nil }
func (f *fakeShiftRepo) FindByID(_ context.Context, id uuid.UUID) (*shift.Shift, error) {
	if s, ok := f.byID[id]; ok {
		return s, nil
	}
	return nil, shift.ErrNotFound
}
func (f *fakeShiftRepo) List(context.Context, pagination.Params) ([]shift.Shift, int64, error) {
	return nil, 0, nil
}
func (f *fakeShiftRepo) Update(context.Context, *shift.Shift) error { return nil }
func (f *fakeShiftRepo) Delete(context.Context, uuid.UUID) error    { return nil }

var _ shift.Repository = (*fakeShiftRepo)(nil)

// fakeScheduleRepo has no overrides configured by default, so resolveShift
// always falls back to the employee's default shift — sufficient for these
// tests, which focus on CheckIn/CheckOut orchestration rather than
// schedule-override resolution (covered separately by internal/schedule).
type fakeScheduleRepo struct{}

func (f *fakeScheduleRepo) Create(context.Context, *schedule.Schedule) error { return nil }
func (f *fakeScheduleRepo) FindByID(context.Context, uuid.UUID) (*schedule.Schedule, error) {
	return nil, schedule.ErrNotFound
}
func (f *fakeScheduleRepo) ListByEmployee(context.Context, uuid.UUID) ([]schedule.Schedule, error) {
	return nil, nil
}
func (f *fakeScheduleRepo) FindForEmployeeAndDay(context.Context, uuid.UUID, int) (*schedule.Schedule, error) {
	return nil, schedule.ErrNotFound
}
func (f *fakeScheduleRepo) List(context.Context, pagination.Params) ([]schedule.Schedule, int64, error) {
	return nil, 0, nil
}
func (f *fakeScheduleRepo) Update(context.Context, *schedule.Schedule) error { return nil }
func (f *fakeScheduleRepo) Delete(context.Context, uuid.UUID) error          { return nil }

var _ schedule.Repository = (*fakeScheduleRepo)(nil)

// fakeAttendanceRepo is a minimal in-memory Repository, enough to exercise
// Service.CheckIn/CheckOut's orchestration and error paths.
type fakeAttendanceRepo struct {
	byID map[uuid.UUID]*Attendance
}

func newFakeAttendanceRepo() *fakeAttendanceRepo {
	return &fakeAttendanceRepo{byID: map[uuid.UUID]*Attendance{}}
}

func (f *fakeAttendanceRepo) Create(_ context.Context, a *Attendance) error {
	for _, existing := range f.byID {
		if existing.EmployeeID == a.EmployeeID && existing.AttendanceDate.Equal(a.AttendanceDate) {
			return ErrDuplicateCheckIn
		}
	}
	a.ID = uuid.New()
	cp := *a
	f.byID[a.ID] = &cp
	return nil
}
func (f *fakeAttendanceRepo) FindByID(_ context.Context, id uuid.UUID) (*Attendance, error) {
	if a, ok := f.byID[id]; ok {
		return a, nil
	}
	return nil, ErrNotFound
}
func (f *fakeAttendanceRepo) FindOpenByEmployee(_ context.Context, employeeID uuid.UUID) (*Attendance, error) {
	for _, a := range f.byID {
		if a.EmployeeID == employeeID && a.CheckOutAt == nil {
			return a, nil
		}
	}
	return nil, ErrNotFound
}
func (f *fakeAttendanceRepo) CompleteCheckOut(_ context.Context, id uuid.UUID, checkOutAt time.Time, deviceID uuid.UUID, workingMinutes int) error {
	a, ok := f.byID[id]
	if !ok || a.CheckOutAt != nil {
		return ErrAlreadyCheckedOut
	}
	a.CheckOutAt = &checkOutAt
	a.CheckOutDeviceID = &deviceID
	a.WorkingDurationMinutes = &workingMinutes
	a.Status = StatusCheckedOut
	return nil
}
func (f *fakeAttendanceRepo) List(context.Context, Filter, pagination.Params) ([]Attendance, int64, error) {
	return nil, 0, nil
}

var _ Repository = (*fakeAttendanceRepo)(nil)

func setupService(t *testing.T) (*Service, uuid.UUID, string) {
	t.Helper()

	empID := uuid.New()
	shiftID := uuid.New()
	emp := &employee.Employee{ID: empID, Status: employee.StatusActive, ShiftID: &shiftID}
	sh := &shift.Shift{ID: shiftID, Name: "Shift Pagi", StartTime: "08:00", LateToleranceMinutes: 10, IsActive: true}
	dev := &device.Device{ID: uuid.New(), DeviceCode: "TAB-001", Status: device.StatusActive}

	svc := NewService(
		newFakeAttendanceRepo(),
		&fakeEmployeeRepo{byID: map[uuid.UUID]*employee.Employee{empID: emp}},
		&fakeDeviceRepo{byCode: map[string]*device.Device{dev.DeviceCode: dev}},
		&fakeShiftRepo{byID: map[uuid.UUID]*shift.Shift{shiftID: sh}},
		&fakeScheduleRepo{},
	)
	return svc, empID, dev.DeviceCode
}

func TestService_CheckIn_Success(t *testing.T) {
	svc, empID, deviceCode := setupService(t)

	att, err := svc.CheckIn(context.Background(), empID, deviceCode)
	if err != nil {
		t.Fatalf("CheckIn() error = %v", err)
	}
	if att.CheckInAt == nil {
		t.Fatal("CheckIn() did not set CheckInAt")
	}
}

func TestService_CheckIn_DuplicateRejected(t *testing.T) {
	svc, empID, deviceCode := setupService(t)

	if _, err := svc.CheckIn(context.Background(), empID, deviceCode); err != nil {
		t.Fatalf("first CheckIn() error = %v", err)
	}
	_, err := svc.CheckIn(context.Background(), empID, deviceCode)
	if !errors.Is(err, ErrAlreadyCheckedIn) {
		t.Fatalf("second CheckIn() error = %v, want ErrAlreadyCheckedIn", err)
	}
}

func TestService_CheckIn_UnknownDeviceRejected(t *testing.T) {
	svc, empID, _ := setupService(t)

	_, err := svc.CheckIn(context.Background(), empID, "NOT-REGISTERED")
	if !errors.Is(err, ErrDeviceNotRegistered) {
		t.Fatalf("CheckIn() error = %v, want ErrDeviceNotRegistered", err)
	}
}

func TestService_CheckIn_UnknownEmployeeRejected(t *testing.T) {
	svc, _, deviceCode := setupService(t)

	_, err := svc.CheckIn(context.Background(), uuid.New(), deviceCode)
	if !errors.Is(err, ErrEmployeeNotFound) {
		t.Fatalf("CheckIn() error = %v, want ErrEmployeeNotFound", err)
	}
}

func TestService_CheckIn_NoShiftAssignedRejected(t *testing.T) {
	empID := uuid.New()
	emp := &employee.Employee{ID: empID, Status: employee.StatusActive, ShiftID: nil}
	dev := &device.Device{ID: uuid.New(), DeviceCode: "TAB-001", Status: device.StatusActive}

	svc := NewService(
		newFakeAttendanceRepo(),
		&fakeEmployeeRepo{byID: map[uuid.UUID]*employee.Employee{empID: emp}},
		&fakeDeviceRepo{byCode: map[string]*device.Device{dev.DeviceCode: dev}},
		&fakeShiftRepo{byID: map[uuid.UUID]*shift.Shift{}},
		&fakeScheduleRepo{},
	)

	_, err := svc.CheckIn(context.Background(), empID, dev.DeviceCode)
	if !errors.Is(err, ErrNoShiftAssigned) {
		t.Fatalf("CheckIn() error = %v, want ErrNoShiftAssigned", err)
	}
}

func TestService_CheckOut_WithoutCheckInRejected(t *testing.T) {
	svc, empID, deviceCode := setupService(t)

	_, err := svc.CheckOut(context.Background(), empID, deviceCode)
	if !errors.Is(err, ErrNoOpenCheckIn) {
		t.Fatalf("CheckOut() error = %v, want ErrNoOpenCheckIn", err)
	}
}

func TestService_CheckIn_ThenCheckOut_ComputesWorkingDuration(t *testing.T) {
	svc, empID, deviceCode := setupService(t)

	if _, err := svc.CheckIn(context.Background(), empID, deviceCode); err != nil {
		t.Fatalf("CheckIn() error = %v", err)
	}

	att, err := svc.CheckOut(context.Background(), empID, deviceCode)
	if err != nil {
		t.Fatalf("CheckOut() error = %v", err)
	}
	if att.Status != StatusCheckedOut {
		t.Errorf("Status = %q, want %q", att.Status, StatusCheckedOut)
	}
	if att.WorkingDurationMinutes == nil {
		t.Fatal("WorkingDurationMinutes is nil after checkout")
	}
	if *att.WorkingDurationMinutes < 0 {
		t.Errorf("WorkingDurationMinutes = %d, want >= 0", *att.WorkingDurationMinutes)
	}

	// A second check-out attempt must fail — there is no open attendance
	// left to close.
	if _, err := svc.CheckOut(context.Background(), empID, deviceCode); !errors.Is(err, ErrNoOpenCheckIn) {
		t.Fatalf("second CheckOut() error = %v, want ErrNoOpenCheckIn", err)
	}
}
