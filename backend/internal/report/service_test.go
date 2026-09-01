package report

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRepo struct {
	employees []EmployeeRow
	overrides map[uuid.UUID]map[int]bool
	company   map[int]bool
	atts      []AttendanceRow
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
