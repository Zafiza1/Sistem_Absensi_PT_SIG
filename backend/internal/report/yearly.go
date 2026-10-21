package report

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Yearly builds the year-end recap — what HR used to assemble by copying
// twelve monthly spreadsheets into one and tallying izin/sakit/alpha/cuti
// by hand. It is the twelve monthly reports summed per employee, plus
// "how many people, how many days" per category for the year and per month.
func (s *Service) Yearly(ctx context.Context, year int, departmentID *uuid.UUID) (*Yearly, error) {
	y := &Yearly{
		Year:        year,
		GeneratedAt: time.Now().In(jakarta),
		Months:      make([]YearMonth, 0, 12),
		Monthly:     make([]*Monthly, 0, 12),
	}

	byID := map[uuid.UUID]*YearlyEmployee{}
	var order []uuid.UUID

	for month := 1; month <= 12; month++ {
		m, err := s.Monthly(ctx, year, month, departmentID)
		if err != nil {
			return nil, err
		}
		y.Monthly = append(y.Monthly, m)

		perMonth := make([]Totals, 0, len(m.Employees))
		for _, e := range m.Employees {
			ye, ok := byID[e.EmployeeID]
			if !ok {
				ye = &YearlyEmployee{
					EmployeeID:     e.EmployeeID,
					EmployeeNumber: e.EmployeeNumber,
					Name:           e.Name,
					DepartmentName: e.DepartmentName,
					Months:         make([]Totals, 12),
				}
				byID[e.EmployeeID] = ye
				order = append(order, e.EmployeeID)
			}
			ye.Months[month-1] = e.Totals
			ye.Totals.add(e.Totals)
			perMonth = append(perMonth, e.Totals)
		}
		y.Months = append(y.Months, YearMonth{Month: month, CategoryCounts: countCategories(perMonth)})
	}

	y.Employees = make([]YearlyEmployee, 0, len(order))
	yearTotals := make([]Totals, 0, len(order))
	for _, id := range order {
		y.Employees = append(y.Employees, *byID[id])
		yearTotals = append(yearTotals, byID[id].Totals)
	}
	y.Summary = countCategories(yearTotals)

	return y, nil
}

// countCategories turns per-employee totals into per-category headcounts:
// People is how many employees had at least one such day, Days the sum.
func countCategories(rows []Totals) CategoryCounts {
	var c CategoryCounts
	tally := func(h *Headcount, days int) {
		if days > 0 {
			h.People++
			h.Days += days
		}
	}
	for _, t := range rows {
		tally(&c.Late, t.LateCount)
		tally(&c.Izin, t.Izin)
		tally(&c.Sakit, t.Sakit)
		tally(&c.Cuti, t.Cuti)
		tally(&c.Absent, t.Absent)
	}
	return c
}
