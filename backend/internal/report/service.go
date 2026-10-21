package report

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidMonth = errors.New("report: month must be between 1 and 12")

// jakarta matches internal/attendance: every date/weekday boundary in the
// report is evaluated in the company's single operating timezone,
// regardless of the server's own.
var jakarta = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}()

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Monthly builds the attendance report for the given year and month (1-12),
// optionally scoped to one department.
func (s *Service) Monthly(ctx context.Context, year, month int, departmentID *uuid.UUID) (*Monthly, error) {
	if month < 1 || month > 12 {
		return nil, ErrInvalidMonth
	}

	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, jakarta)
	daysInMonth := first.AddDate(0, 1, -1).Day()
	last := time.Date(year, time.Month(month), daysInMonth, 0, 0, 0, 0, jakarta)
	today := dateOnly(time.Now().In(jakarta))

	employees, err := s.repo.ActiveEmployees(ctx, departmentID)
	if err != nil {
		return nil, err
	}
	overrides, err := s.repo.ScheduleOverrideDays(ctx)
	if err != nil {
		return nil, err
	}
	company, err := s.repo.CompanyScheduleDays(ctx)
	if err != nil {
		return nil, err
	}
	atts, err := s.repo.AttendanceInRange(ctx, first, last, departmentID)
	if err != nil {
		return nil, err
	}
	leaves, err := s.repo.LeavesInRange(ctx, first, last, departmentID)
	if err != nil {
		return nil, err
	}
	leaveByEmpDay := leaveDaysInMonth(leaves, year, time.Month(month))

	// (employee, day-of-month) -> attendance
	attByEmpDay := make(map[uuid.UUID]map[int]AttendanceRow, len(employees))
	for _, a := range atts {
		if attByEmpDay[a.EmployeeID] == nil {
			attByEmpDay[a.EmployeeID] = map[int]AttendanceRow{}
		}
		attByEmpDay[a.EmployeeID][a.Date.Day()] = a
	}

	report := &Monthly{
		Year:        year,
		Month:       month,
		DaysInMonth: daysInMonth,
		GeneratedAt: time.Now().In(jakarta),
		Employees:   make([]EmployeeReport, 0, len(employees)),
	}

	for _, emp := range employees {
		er := EmployeeReport{
			EmployeeID:     emp.ID,
			EmployeeNumber: emp.EmployeeNumber,
			Name:           emp.Name,
			DepartmentName: emp.DepartmentName,
			Days:           make([]DayCell, 0, daysInMonth),
		}

		for day := 1; day <= daysInMonth; day++ {
			d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, jakarta)
			iso := isoWeekday(d)
			working := isWorkingDay(emp.ShiftID, overrides[emp.ID][iso], company, iso)

			cell := DayCell{Day: day}
			att, hasAtt := attByEmpDay[emp.ID][day]
			leaveType, onLeave := leaveByEmpDay[emp.ID][day]

			switch {
			case hasAtt:
				cell.CheckInAt = att.CheckInAt
				cell.CheckOutAt = att.CheckOutAt
				cell.LateMinutes = att.LateMinutes
				er.WorkingDays++
				if att.LateMinutes > 0 {
					cell.Status = DayLate
					er.LateCount++
					er.LateMinutes += att.LateMinutes
				} else {
					cell.Status = DayOnTime
					er.OnTime++
				}
			case !working:
				cell.Status = DayOff
			case onLeave:
				// Counted even when still ahead: HR recorded it, so it's
				// no longer pending.
				cell.Status = DayStatus(leaveType)
				er.WorkingDays++
				switch cell.Status {
				case DayIzin:
					er.Izin++
				case DaySakit:
					er.Sakit++
				default:
					cell.Status = DayCuti
					er.Cuti++
				}
			case !d.Before(today):
				cell.Status = DayPending
			default:
				cell.Status = DayAbsent
				er.WorkingDays++
				er.Absent++
			}

			er.Days = append(er.Days, cell)
		}

		report.Employees = append(report.Employees, er)
	}

	return report, nil
}

// leaveDaysInMonth expands each leave period into the days of the given
// month it covers: employee -> day-of-month -> leave type. Leaves are
// prevented from overlapping by internal/leave, so each day has at most one.
func leaveDaysInMonth(leaves []LeaveRow, year int, month time.Month) map[uuid.UUID]map[int]string {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1)

	out := map[uuid.UUID]map[int]string{}
	for _, l := range leaves {
		start := time.Date(l.StartDate.Year(), l.StartDate.Month(), l.StartDate.Day(), 0, 0, 0, 0, time.UTC)
		end := time.Date(l.EndDate.Year(), l.EndDate.Month(), l.EndDate.Day(), 0, 0, 0, 0, time.UTC)
		if start.Before(first) {
			start = first
		}
		if end.After(last) {
			end = last
		}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			if out[l.EmployeeID] == nil {
				out[l.EmployeeID] = map[int]string{}
			}
			out[l.EmployeeID][d.Day()] = l.Type
		}
	}
	return out
}

// isWorkingDay mirrors attendance.resolveShift's three layers, reduced to
// the single question the report asks: was this a day the employee was
// expected to come in?
func isWorkingDay(empShiftID *uuid.UUID, hasOverride bool, company map[int]bool, iso int) bool {
	if hasOverride {
		return true
	}
	if working, configured := company[iso]; configured {
		return working
	}
	return empShiftID != nil
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// isoWeekday converts Go's Weekday (Sunday=0) to ISO 8601 (Monday=1..Sunday=7),
// matching work_schedules.day_of_week and company_schedules.day_of_week.
func isoWeekday(t time.Time) int {
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}
