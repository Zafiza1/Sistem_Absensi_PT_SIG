package attendance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/companyschedule"
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

// TestComputeCheckInStatus_OvernightShift covers the graveyard shift
// (22:00 -> 06:00): check-ins the same evening are measured against tonight's
// 22:00, while check-ins after midnight are measured against *last* night's
// 22:00 rather than being treated as ~21h early and wrongly marked on-time.
func TestComputeCheckInStatus_OvernightShift(t *testing.T) {
	sh := &shift.Shift{StartTime: "22:00", EndTime: "06:00", IsOvernight: true, LateToleranceMinutes: 10}

	tests := []struct {
		name       string
		now        time.Time
		wantStatus string
		wantLate   int
	}{
		{"ten minutes before start", time.Date(2026, 1, 5, 21, 50, 0, 0, jakarta), StatusOnTime, 0},
		{"exactly on time", time.Date(2026, 1, 5, 22, 0, 0, 0, jakarta), StatusOnTime, 0},
		{"within tolerance", time.Date(2026, 1, 5, 22, 9, 0, 0, jakarta), StatusOnTime, 0},
		{"late, still before midnight", time.Date(2026, 1, 5, 22, 45, 0, 0, jakarta), StatusLate, 45},
		{"late, just after midnight", time.Date(2026, 1, 6, 0, 30, 0, 0, jakarta), StatusLate, 150},
		{"very late, hours after midnight", time.Date(2026, 1, 6, 2, 0, 0, 0, jakarta), StatusLate, 240},
		{"arriving at shift end", time.Date(2026, 1, 6, 6, 0, 0, 0, jakarta), StatusLate, 480},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, late := computeCheckInStatus(tt.now, sh)
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

// fakeScheduleRepo resolves per-weekday overrides from byDay (ISO weekday
// 1..7 -> schedule). A nil/empty map means no overrides, so resolveShift
// always falls back to the employee's default shift.
type fakeScheduleRepo struct {
	byDay map[int]*schedule.Schedule
}

func (f *fakeScheduleRepo) Create(context.Context, *schedule.Schedule) error { return nil }
func (f *fakeScheduleRepo) FindByID(context.Context, uuid.UUID) (*schedule.Schedule, error) {
	return nil, schedule.ErrNotFound
}
func (f *fakeScheduleRepo) ListByEmployee(context.Context, uuid.UUID) ([]schedule.Schedule, error) {
	return nil, nil
}
func (f *fakeScheduleRepo) FindForEmployeeAndDay(_ context.Context, _ uuid.UUID, day int) (*schedule.Schedule, error) {
	if sc, ok := f.byDay[day]; ok {
		return sc, nil
	}
	return nil, schedule.ErrNotFound
}
func (f *fakeScheduleRepo) List(context.Context, pagination.Params) ([]schedule.Schedule, int64, error) {
	return nil, 0, nil
}
func (f *fakeScheduleRepo) Update(context.Context, *schedule.Schedule) error { return nil }
func (f *fakeScheduleRepo) Delete(context.Context, uuid.UUID) error          { return nil }

var _ schedule.Repository = (*fakeScheduleRepo)(nil)

// fakeCompanyScheduleRepo resolves the company-wide weekly default from
// byDay (ISO weekday 1..7 -> Day). A day absent from the map is
// ErrNotFound (not configured); a day present with a nil ShiftID is a
// configured non-working day.
type fakeCompanyScheduleRepo struct {
	byDay map[int]companyschedule.Day
}

func (f *fakeCompanyScheduleRepo) List(context.Context) ([]companyschedule.Day, error) {
	out := make([]companyschedule.Day, 0, len(f.byDay))
	for _, d := range f.byDay {
		out = append(out, d)
	}
	return out, nil
}
func (f *fakeCompanyScheduleRepo) FindByDay(_ context.Context, day int) (*companyschedule.Day, error) {
	if d, ok := f.byDay[day]; ok {
		return &d, nil
	}
	return nil, companyschedule.ErrNotFound
}
func (f *fakeCompanyScheduleRepo) Replace(context.Context, []companyschedule.Day) error { return nil }

var _ companyschedule.Repository = (*fakeCompanyScheduleRepo)(nil)

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
		&fakeCompanyScheduleRepo{},
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
		&fakeCompanyScheduleRepo{},
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

// --- Schedule override resolution (Phase 7) ---

// setupServiceWithSchedule builds a service where the employee has a default
// shift plus an optional per-weekday override. overrideDay of 0 means "no
// override configured".
func setupServiceWithSchedule(t *testing.T, overrideDay int) (svc *Service, empID uuid.UUID, deviceCode string, defaultShiftID, overrideShiftID uuid.UUID) {
	t.Helper()

	empID = uuid.New()
	defaultShiftID = uuid.New()
	overrideShiftID = uuid.New()

	emp := &employee.Employee{ID: empID, Status: employee.StatusActive, ShiftID: &defaultShiftID}
	defaultShift := &shift.Shift{ID: defaultShiftID, Name: "Shift Pagi", StartTime: "08:00", EndTime: "16:00", LateToleranceMinutes: 10, IsActive: true}
	overrideShift := &shift.Shift{ID: overrideShiftID, Name: "Shift Malam", StartTime: "22:00", EndTime: "06:00", IsOvernight: true, LateToleranceMinutes: 10, IsActive: true}
	dev := &device.Device{ID: uuid.New(), DeviceCode: "TAB-001", Status: device.StatusActive}

	sched := &fakeScheduleRepo{}
	if overrideDay != 0 {
		sched.byDay = map[int]*schedule.Schedule{
			overrideDay: {EmployeeID: empID, ShiftID: overrideShiftID, DayOfWeek: overrideDay},
		}
	}

	svc = NewService(
		newFakeAttendanceRepo(),
		&fakeEmployeeRepo{byID: map[uuid.UUID]*employee.Employee{empID: emp}},
		&fakeDeviceRepo{byCode: map[string]*device.Device{dev.DeviceCode: dev}},
		&fakeShiftRepo{byID: map[uuid.UUID]*shift.Shift{defaultShiftID: defaultShift, overrideShiftID: overrideShift}},
		sched,
		&fakeCompanyScheduleRepo{},
	)
	return svc, empID, dev.DeviceCode, defaultShiftID, overrideShiftID
}

func TestService_CheckIn_UsesScheduleOverrideForToday(t *testing.T) {
	today := isoWeekday(time.Now().In(jakarta))
	svc, empID, deviceCode, _, overrideShiftID := setupServiceWithSchedule(t, today)

	att, err := svc.CheckIn(context.Background(), empID, deviceCode)
	if err != nil {
		t.Fatalf("CheckIn() error = %v", err)
	}
	if att.ShiftID == nil || *att.ShiftID != overrideShiftID {
		t.Fatalf("attendance ShiftID = %v, want schedule-override shift %v", att.ShiftID, overrideShiftID)
	}
}

func TestService_CheckIn_FallsBackToDefaultShiftWhenOverrideIsForAnotherDay(t *testing.T) {
	today := isoWeekday(time.Now().In(jakarta))
	otherDay := today%7 + 1 // any weekday that isn't today
	svc, empID, deviceCode, defaultShiftID, _ := setupServiceWithSchedule(t, otherDay)

	att, err := svc.CheckIn(context.Background(), empID, deviceCode)
	if err != nil {
		t.Fatalf("CheckIn() error = %v", err)
	}
	if att.ShiftID == nil || *att.ShiftID != defaultShiftID {
		t.Fatalf("attendance ShiftID = %v, want default shift %v", att.ShiftID, defaultShiftID)
	}
}

// TestService_ResolveShift_CompanyWeeklySchedule mirrors PT Surya Inti Gas's
// actual pattern: a default 08:00-16:00 shift for Mon-Fri, and a Saturday
// (ISO day 6) work_schedules override to 08:00-14:00. Sunday has no row, so
// it falls back to the default shift.
func TestService_ResolveShift_CompanyWeeklySchedule(t *testing.T) {
	empID := uuid.New()
	weekdayShiftID := uuid.New()
	saturdayShiftID := uuid.New()

	emp := &employee.Employee{ID: empID, Status: employee.StatusActive, ShiftID: &weekdayShiftID}
	weekdayShift := &shift.Shift{ID: weekdayShiftID, Name: "Reguler", StartTime: "08:00", EndTime: "16:00", LateToleranceMinutes: 10, IsActive: true}
	saturdayShift := &shift.Shift{ID: saturdayShiftID, Name: "Sabtu", StartTime: "08:00", EndTime: "14:00", LateToleranceMinutes: 10, IsActive: true}

	svc := NewService(
		newFakeAttendanceRepo(),
		&fakeEmployeeRepo{byID: map[uuid.UUID]*employee.Employee{empID: emp}},
		&fakeDeviceRepo{},
		&fakeShiftRepo{byID: map[uuid.UUID]*shift.Shift{weekdayShiftID: weekdayShift, saturdayShiftID: saturdayShift}},
		&fakeScheduleRepo{byDay: map[int]*schedule.Schedule{
			6: {EmployeeID: empID, ShiftID: saturdayShiftID, DayOfWeek: 6},
		}},
		&fakeCompanyScheduleRepo{},
	)

	for day := 1; day <= 7; day++ {
		want := weekdayShiftID
		if day == 6 {
			want = saturdayShiftID
		}
		sh, err := svc.resolveShift(context.Background(), emp, day)
		if err != nil {
			t.Fatalf("resolveShift(day=%d) error = %v", day, err)
		}
		if sh.ID != want {
			t.Errorf("resolveShift(day=%d) = %q (%v), want %v", day, sh.Name, sh.ID, want)
		}
	}
}

// An employee with no default shift can still check in on a day that has a
// schedule override — the override is consulted before employees.shift_id.
func TestService_CheckIn_ScheduleOverrideCoversMissingDefaultShift(t *testing.T) {
	today := isoWeekday(time.Now().In(jakarta))
	empID := uuid.New()
	overrideShiftID := uuid.New()
	emp := &employee.Employee{ID: empID, Status: employee.StatusActive, ShiftID: nil}
	overrideShift := &shift.Shift{ID: overrideShiftID, Name: "Shift Malam", StartTime: "22:00", EndTime: "06:00", IsOvernight: true, LateToleranceMinutes: 10, IsActive: true}
	dev := &device.Device{ID: uuid.New(), DeviceCode: "TAB-001", Status: device.StatusActive}

	svc := NewService(
		newFakeAttendanceRepo(),
		&fakeEmployeeRepo{byID: map[uuid.UUID]*employee.Employee{empID: emp}},
		&fakeDeviceRepo{byCode: map[string]*device.Device{dev.DeviceCode: dev}},
		&fakeShiftRepo{byID: map[uuid.UUID]*shift.Shift{overrideShiftID: overrideShift}},
		&fakeScheduleRepo{byDay: map[int]*schedule.Schedule{today: {EmployeeID: empID, ShiftID: overrideShiftID, DayOfWeek: today}}},
		&fakeCompanyScheduleRepo{},
	)

	att, err := svc.CheckIn(context.Background(), empID, dev.DeviceCode)
	if err != nil {
		t.Fatalf("CheckIn() error = %v", err)
	}
	if att.ShiftID == nil || *att.ShiftID != overrideShiftID {
		t.Fatalf("attendance ShiftID = %v, want %v", att.ShiftID, overrideShiftID)
	}
}

// --- Company-wide default schedule resolution (Phase 7) ---

// resolveShift priority: per-employee override > company schedule >
// employee default shift; a company-schedule day with no shift is ErrDayOff.
func TestService_ResolveShift_CompanySchedulePriority(t *testing.T) {
	empID := uuid.New()
	empDefaultID := uuid.New()
	companyID := uuid.New()
	overrideID := uuid.New()

	emp := &employee.Employee{ID: empID, Status: employee.StatusActive, ShiftID: &empDefaultID}
	shifts := map[uuid.UUID]*shift.Shift{
		empDefaultID: {ID: empDefaultID, Name: "Default Karyawan", StartTime: "09:00", EndTime: "17:00"},
		companyID:    {ID: companyID, Name: "Reguler", StartTime: "08:00", EndTime: "16:00"},
		overrideID:   {ID: overrideID, Name: "Override", StartTime: "07:00", EndTime: "15:00"},
	}

	build := func(sched *fakeScheduleRepo, company *fakeCompanyScheduleRepo) *Service {
		return NewService(
			newFakeAttendanceRepo(),
			&fakeEmployeeRepo{byID: map[uuid.UUID]*employee.Employee{empID: emp}},
			&fakeDeviceRepo{},
			&fakeShiftRepo{byID: shifts},
			sched,
			company,
		)
	}

	companyMonFri := &fakeCompanyScheduleRepo{byDay: map[int]companyschedule.Day{
		1: {DayOfWeek: 1, ShiftID: &companyID},
		7: {DayOfWeek: 7}, // Sunday configured as a non-working day
	}}

	t.Run("company schedule governs when no per-employee override", func(t *testing.T) {
		sh, err := build(&fakeScheduleRepo{}, companyMonFri).resolveShift(context.Background(), emp, 1)
		if err != nil {
			t.Fatalf("resolveShift() error = %v", err)
		}
		if sh.ID != companyID {
			t.Errorf("resolved %q, want company shift", sh.Name)
		}
	})

	t.Run("per-employee override beats company schedule", func(t *testing.T) {
		sched := &fakeScheduleRepo{byDay: map[int]*schedule.Schedule{1: {EmployeeID: empID, ShiftID: overrideID, DayOfWeek: 1}}}
		sh, err := build(sched, companyMonFri).resolveShift(context.Background(), emp, 1)
		if err != nil {
			t.Fatalf("resolveShift() error = %v", err)
		}
		if sh.ID != overrideID {
			t.Errorf("resolved %q, want override shift", sh.Name)
		}
	})

	t.Run("company-schedule non-working day is ErrDayOff", func(t *testing.T) {
		_, err := build(&fakeScheduleRepo{}, companyMonFri).resolveShift(context.Background(), emp, 7)
		if !errors.Is(err, ErrDayOff) {
			t.Fatalf("resolveShift() error = %v, want ErrDayOff", err)
		}
	})

	t.Run("but a per-employee override still lets them work a company day off", func(t *testing.T) {
		sched := &fakeScheduleRepo{byDay: map[int]*schedule.Schedule{7: {EmployeeID: empID, ShiftID: overrideID, DayOfWeek: 7}}}
		sh, err := build(sched, companyMonFri).resolveShift(context.Background(), emp, 7)
		if err != nil {
			t.Fatalf("resolveShift() error = %v", err)
		}
		if sh.ID != overrideID {
			t.Errorf("resolved %q, want override shift", sh.Name)
		}
	})

	t.Run("unconfigured company day falls through to employee default", func(t *testing.T) {
		sh, err := build(&fakeScheduleRepo{}, companyMonFri).resolveShift(context.Background(), emp, 3)
		if err != nil {
			t.Fatalf("resolveShift() error = %v", err)
		}
		if sh.ID != empDefaultID {
			t.Errorf("resolved %q, want employee default shift", sh.Name)
		}
	})
}
