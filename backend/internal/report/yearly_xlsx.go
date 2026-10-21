package report

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/xuri/excelize/v2"
)

const sheetPerEmployee = "Per Karyawan"

// monthSheetNamesID are the yearly workbook's per-month sheet names.
var monthSheetNamesID = [...]string{
	"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun",
	"Jul", "Agu", "Sep", "Okt", "Nov", "Des",
}

// BuildYearlyXLSX renders the year-end recap: a Ringkasan sheet answering
// "how many people were izin/sakit/alpha/cuti, for how many days" for the
// year and per month, a Per Karyawan sheet of yearly totals, one day-grid
// sheet per month (Jan..Des) to check against, and a legend.
func BuildYearlyXLSX(y *Yearly) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetSheetName("Sheet1", sheetSummary)
	sheets := []string{sheetPerEmployee}
	for m := 1; m <= 12; m++ {
		sheets = append(sheets, monthSheetNamesID[m])
	}
	sheets = append(sheets, sheetLegend)
	for _, name := range sheets {
		if _, err := f.NewSheet(name); err != nil {
			return nil, err
		}
	}

	header, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E2E8F0"}},
	})
	title, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	fl := newFills(f)

	writeYearlySummarySheet(f, y, header, title)

	rows := make([]employeeTotals, 0, len(y.Employees))
	for _, e := range y.Employees {
		rows = append(rows, employeeTotals{e.EmployeeNumber, e.Name, e.DepartmentName, e.Totals})
	}
	writeTotalsTable(f, sheetPerEmployee, header, rows)

	for _, m := range y.Monthly {
		writeDayGrid(f, monthSheetNamesID[m.Month], m, header, fl, true)
	}

	writeLegend(f, header, [][2]string{
		{"Laporan", fmt.Sprintf("Rekap Kehadiran Tahun %d", y.Year)},
		{"Dibuat", y.GeneratedAt.Format("2 January 2006 15:04") + " WIB"},
		{"Jumlah karyawan", strconv.Itoa(len(y.Employees))},
	}, "Arti (sheet Jan s/d Des)")

	f.SetActiveSheet(0)
	return f.WriteToBuffer()
}

// yearlyCategories are the Ringkasan rows/column groups, in order.
var yearlyCategories = []struct {
	label string
	get   func(CategoryCounts) Headcount
}{
	{"Izin", func(c CategoryCounts) Headcount { return c.Izin }},
	{"Sakit", func(c CategoryCounts) Headcount { return c.Sakit }},
	{"Tidak Hadir (alpha)", func(c CategoryCounts) Headcount { return c.Absent }},
	{"Cuti", func(c CategoryCounts) Headcount { return c.Cuti }},
	{"Terlambat", func(c CategoryCounts) Headcount { return c.Late }},
}

func writeYearlySummarySheet(f *excelize.File, y *Yearly, headerStyle, titleStyle int) {
	const sh = sheetSummary
	set := func(col, row int, v any) {
		cell, _ := excelize.CoordinatesToCellName(col, row)
		f.SetCellValue(sh, cell, v)
	}
	styleRow := func(row, lastCol int) {
		from, _ := excelize.CoordinatesToCellName(1, row)
		to, _ := excelize.CoordinatesToCellName(lastCol, row)
		f.SetCellStyle(sh, from, to, headerStyle)
	}

	f.SetColWidth(sh, "A", "A", 22)
	f.SetColWidth(sh, "B", "K", 13)
	set(1, 1, fmt.Sprintf("Rekap Kehadiran Tahun %d", y.Year))
	f.SetCellStyle(sh, "A1", "A1", titleStyle)
	set(1, 2, fmt.Sprintf("%d karyawan aktif", len(y.Employees)))

	// Year totals: one row per category.
	row := 4
	set(1, row, "Kategori")
	set(2, row, "Jumlah Orang")
	set(3, row, "Jumlah Hari")
	styleRow(row, 3)
	for _, c := range yearlyCategories {
		row++
		h := c.get(y.Summary)
		set(1, row, c.label)
		set(2, row, h.People)
		set(3, row, h.Days)
	}

	// Per month: Bulan, then (orang, hari) per category.
	row += 3
	set(1, row, "Per Bulan")
	f.SetCellStyle(sh, "A"+strconv.Itoa(row), "A"+strconv.Itoa(row), titleStyle)
	row++
	set(1, row, "Bulan")
	for i, c := range yearlyCategories {
		set(2+2*i, row, c.label+" (orang)")
		set(3+2*i, row, c.label+" (hari)")
	}
	lastCol := 1 + 2*len(yearlyCategories)
	styleRow(row, lastCol)
	for _, m := range y.Months {
		row++
		set(1, row, monthNamesID[m.Month])
		for i, c := range yearlyCategories {
			h := c.get(m.CategoryCounts)
			set(2+2*i, row, h.People)
			set(3+2*i, row, h.Days)
		}
	}
}

// YearlyFileName is the download filename for a year's recap.
func YearlyFileName(year int) string {
	return fmt.Sprintf("rekap-kehadiran-%04d.xlsx", year)
}
