package report

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

type fakeRepo struct {
	employees []EmployeeRow
	overrides map[uuid.UUID]map[int]bool
	company   map[int]bool
	atts      []AttendanceRow
	leaves    []LeaveRow
}

func (f *fakeRepo) ActiveEmployees(context.Context, *uuid.UUID) ([]EmployeeRow, error) {
	return f.employees, nil
}
func (f *fakeRepo) ScheduleOverrideDays(context.Context) (map[uuid.UUID]map[int]bool, error) {
	return f.overrides, nil
}
func (f *fakeRepo) CompanyScheduleDays(context.Context) (map[int]bool, error) { return f.company, nil }
func (f *fakeRepo) AttendanceInRange(context.Context, time.Time, time.Time, *uuid.UUID) ([]AttendanceRow, error) {
	return f.atts, nil
}

func (f *fakeRepo) LeavesInRange(_ context.Context, from, to time.Time, _ *uuid.UUID) ([]LeaveRow, error) {
	var out []LeaveRow
	for _, l := range f.leaves {
		if !l.StartDate.After(to) && !l.EndDate.Before(from) {
			out = append(out, l)
		}
	}
	return out, nil
}

func date(month, day int) time.Time { return time.Date(2026, time.Month(month), day, 0, 0, 0, 0, time.UTC) }

func att(empID uuid.UUID, day, late int) AttendanceRow {
	t := time.Date(2026, 1, day, 8, 0, 0, 0, jakarta)
	return AttendanceRow{EmployeeID: empID, Date: time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC), CheckInAt: &t, LateMinutes: late}
}

func TestService_Monthly_Aggregates(t *testing.T) {
	empID := uuid.New()
	repo := &fakeRepo{
		employees: []EmployeeRow{{ID: empID, EmployeeNumber: "1001", Name: "Budi"}},
		// Mon-Sat working, Sunday off.
		company: map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: false},
		atts: []AttendanceRow{
			att(empID, 2, 0),  // Fri 2 Jan — on time
			att(empID, 5, 20), // Mon 5 Jan — late 20
		},
	}

	r, err := NewService(repo).Monthly(context.Background(), 2026, 1, nil)
	if err != nil {
		t.Fatalf("Monthly() error = %v", err)
	}

	// January 2026 has 31 days and 4 Sundays (4, 11, 18, 25) => 27 working days.
	e := r.Employees[0]
	if e.WorkingDays != 27 {
		t.Errorf("WorkingDays = %d, want 27", e.WorkingDays)
	}
	if e.OnTime != 1 {
		t.Errorf("OnTime = %d, want 1", e.OnTime)
	}
	if e.LateCount != 1 || e.LateMinutes != 20 {
		t.Errorf("Late = %d count / %d min, want 1 / 20", e.LateCount, e.LateMinutes)
	}
	if e.Absent != 25 {
		t.Errorf("Absent = %d, want 25", e.Absent)
	}
	if got := e.OnTime + e.LateCount + e.Absent; got != e.WorkingDays {
		t.Errorf("OnTime+Late+Absent = %d, want == WorkingDays %d", got, e.WorkingDays)
	}

	// Sundays must be OFF, not ABSENT.
	if e.Days[3].Status != DayOff { // 4 Jan 2026 is a Sunday
		t.Errorf("4 Jan status = %q, want OFF", e.Days[3].Status)
	}
	if e.Days[1].Status != DayOnTime { // 2 Jan
		t.Errorf("2 Jan status = %q, want ON_TIME", e.Days[1].Status)
	}
	if e.Days[4].Status != DayLate { // 5 Jan
		t.Errorf("5 Jan status = %q, want LATE", e.Days[4].Status)
	}
}

func TestService_Monthly_LeaveIsNotAbsent(t *testing.T) {
	empID := uuid.New()
	repo := &fakeRepo{
		employees: []EmployeeRow{{ID: empID, EmployeeNumber: "1001", Name: "Budi"}},
		company:   map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: false},
		atts:      []AttendanceRow{att(empID, 2, 0)},
		leaves: []LeaveRow{
			{EmployeeID: empID, Type: "IZIN", StartDate: date(1, 5), EndDate: date(1, 5)},
			{EmployeeID: empID, Type: "SAKIT", StartDate: date(1, 6), EndDate: date(1, 7)},
			// Spans into February: only 30-31 Jan count here.
			{EmployeeID: empID, Type: "CUTI", StartDate: date(1, 30), EndDate: date(2, 3)},
			// Checked in on the 2nd anyway: attendance wins over the leave.
			{EmployeeID: empID, Type: "IZIN", StartDate: date(1, 2), EndDate: date(1, 2)},
		},
	}

	r, err := NewService(repo).Monthly(context.Background(), 2026, 1, nil)
	if err != nil {
		t.Fatalf("Monthly() error = %v", err)
	}
	e := r.Employees[0]
	if e.Izin != 1 || e.Sakit != 2 || e.Cuti != 2 || e.OnTime != 1 {
		t.Errorf("izin/sakit/cuti/on_time = %d/%d/%d/%d, want 1/2/2/1", e.Izin, e.Sakit, e.Cuti, e.OnTime)
	}
	if e.Absent != 27-1-1-2-2 {
		t.Errorf("Absent = %d, want %d", e.Absent, 27-6)
	}
	if e.WorkingDays != 27 {
		t.Errorf("WorkingDays = %d, want 27", e.WorkingDays)
	}
	if e.Days[4].Status != DayIzin || e.Days[29].Status != DayCuti {
		t.Errorf("5 Jan / 30 Jan = %q / %q, want IZIN / CUTI", e.Days[4].Status, e.Days[29].Status)
	}
}

func TestService_Yearly_Headcounts(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	repo := &fakeRepo{
		employees: []EmployeeRow{{ID: a, EmployeeNumber: "1", Name: "A"}, {ID: b, EmployeeNumber: "2", Name: "B"}},
		company:   map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: false},
		leaves: []LeaveRow{
			{EmployeeID: a, Type: "SAKIT", StartDate: date(1, 5), EndDate: date(1, 6)},
			{EmployeeID: a, Type: "SAKIT", StartDate: date(3, 2), EndDate: date(3, 2)},
			{EmployeeID: b, Type: "SAKIT", StartDate: date(3, 3), EndDate: date(3, 3)},
		},
	}

	y, err := NewService(repo).Yearly(context.Background(), 2026, nil)
	if err != nil {
		t.Fatalf("Yearly() error = %v", err)
	}
	if y.Summary.Sakit != (Headcount{People: 2, Days: 4}) {
		t.Errorf("Summary.Sakit = %+v, want 2 people / 4 days", y.Summary.Sakit)
	}
	if y.Months[2].Sakit != (Headcount{People: 2, Days: 2}) {
		t.Errorf("March Sakit = %+v, want 2 people / 2 days", y.Months[2].Sakit)
	}
	if y.Employees[0].Sakit != 3 || y.Employees[0].Months[0].Sakit != 2 {
		t.Errorf("A sakit year/jan = %d/%d, want 3/2", y.Employees[0].Sakit, y.Employees[0].Months[0].Sakit)
	}
}

func TestService_Monthly_FutureDaysArePending(t *testing.T) {
	// A month that contains "today" (per the test clock, 2026-09): future
	// working days must be PENDING, never ABSENT.
	now := time.Now().In(jakarta)
	empID := uuid.New()
	repo := &fakeRepo{
		employees: []EmployeeRow{{ID: empID, EmployeeNumber: "1", Name: "A"}},
		company:   map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true},
	}

	r, err := NewService(repo).Monthly(context.Background(), now.Year(), int(now.Month()), nil)
	if err != nil {
		t.Fatalf("Monthly() error = %v", err)
	}
	e := r.Employees[0]
	for _, d := range e.Days {
		day := time.Date(now.Year(), now.Month(), d.Day, 0, 0, 0, 0, jakarta)
		if day.After(dateOnly(now)) && d.Status != DayPending {
			t.Fatalf("day %d (future) status = %q, want PENDING", d.Day, d.Status)
		}
	}
}

func TestService_Monthly_RejectsBadMonth(t *testing.T) {
	_, err := NewService(&fakeRepo{}).Monthly(context.Background(), 2026, 13, nil)
	if err != ErrInvalidMonth {
		t.Fatalf("Monthly() error = %v, want ErrInvalidMonth", err)
	}
}

func TestBuildXLSX_Smoke(t *testing.T) {
	empID := uuid.New()
	repo := &fakeRepo{
		employees: []EmployeeRow{{ID: empID, EmployeeNumber: "1001", Name: "Budi", DepartmentName: "Produksi"}},
		company:   map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: false},
		atts:      []AttendanceRow{att(empID, 5, 20)},
	}
	r, _ := NewService(repo).Monthly(context.Background(), 2026, 1, nil)

	buf, err := BuildXLSX(r)
	if err != nil {
		t.Fatalf("BuildXLSX() error = %v", err)
	}
	if buf.Len() < 1000 {
		t.Errorf("xlsx buffer = %d bytes, looks too small", buf.Len())
	}
}

func TestBuildYearlyXLSX_Smoke(t *testing.T) {
	empID := uuid.New()
	repo := &fakeRepo{
		employees: []EmployeeRow{{ID: empID, EmployeeNumber: "1001", Name: "Budi"}},
		company:   map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: false},
		leaves:    []LeaveRow{{EmployeeID: empID, Type: "SAKIT", StartDate: date(2, 2), EndDate: date(2, 3)}},
	}
	y, err := NewService(repo).Yearly(context.Background(), 2026, nil)
	if err != nil {
		t.Fatalf("Yearly() error = %v", err)
	}
	buf, err := BuildYearlyXLSX(y)
	if err != nil {
		t.Fatalf("BuildYearlyXLSX() error = %v", err)
	}
	f, err := excelize.OpenReader(buf)
	if err != nil {
		t.Fatalf("reopen xlsx: %v", err)
	}
	if got := len(f.GetSheetList()); got != 15 {
		t.Errorf("sheets = %d, want 15 (Ringkasan, Per Karyawan, 12 months, Keterangan)", got)
	}
	if v, _ := f.GetCellValue("Feb", "D2"); v != "S" { // column D = 2 Feb 2026, a Monday
		t.Errorf("Feb!D2 = %q, want S", v)
	}
}
