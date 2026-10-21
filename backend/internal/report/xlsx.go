package report

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/xuri/excelize/v2"
)

var monthNamesID = [...]string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

const (
	sheetSummary = "Ringkasan"
	sheetDetail  = "Detail Harian"
	sheetLegend  = "Keterangan"
)

// BuildXLSX renders the monthly report as a three-sheet .xlsx workbook:
// a per-employee summary, a day-by-day grid, and a legend.
func BuildXLSX(r *Monthly) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetSheetName("Sheet1", sheetSummary)
	if _, err := f.NewSheet(sheetDetail); err != nil {
		return nil, err
	}
	if _, err := f.NewSheet(sheetLegend); err != nil {
		return nil, err
	}

	header, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E2E8F0"}},
	})
	fills := newFills(f)

	writeSummarySheet(f, r, header)
	writeDetailSheet(f, r, header, fills)
	writeLegendSheet(f, r, header)

	f.SetActiveSheet(0)
	return f.WriteToBuffer()
}

// fills are the day-cell background styles shared by every grid sheet.
type fills struct {
	late, absent, off, izin, sakit, cuti int
}

func newFills(f *excelize.File) fills {
	fill := func(color string) int {
		id, _ := f.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{color}}})
		return id
	}
	return fills{
		late:   fill("FEF3C7"),
		absent: fill("FEE2E2"),
		off:    fill("F1F5F9"),
		izin:   fill("DBEAFE"),
		sakit:  fill("EDE9FE"),
		cuti:   fill("D1FAE5"),
	}
}

// totalsColumns are the per-employee count columns, in order, after NIK,
// Nama and Divisi.
var totalsColumns = []struct {
	head  string
	width float64
	value func(Totals) int
}{
	{"Hari Kerja", 11, func(t Totals) int { return t.WorkingDays }},
	{"Tepat Waktu", 12, func(t Totals) int { return t.OnTime }},
	{"Terlambat (kali)", 15, func(t Totals) int { return t.LateCount }},
	{"Total Menit Terlambat", 20, func(t Totals) int { return t.LateMinutes }},
	{"Izin", 8, func(t Totals) int { return t.Izin }},
	{"Sakit", 8, func(t Totals) int { return t.Sakit }},
	{"Cuti", 8, func(t Totals) int { return t.Cuti }},
	{"Tidak Hadir", 12, func(t Totals) int { return t.Absent }},
}

// writeTotalsTable writes a NIK/Nama/Divisi + totalsColumns table starting
// at row 1 of sheet, one row per employee.
func writeTotalsTable(f *excelize.File, sheet string, headerStyle int, rows []employeeTotals) {
	heads := []struct {
		head  string
		width float64
	}{{"NIK", 14}, {"Nama", 28}, {"Divisi", 20}}
	for _, c := range totalsColumns {
		heads = append(heads, struct {
			head  string
			width float64
		}{c.head, c.width})
	}
	for i, c := range heads {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, c.width)
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, c.head)
	}
	lastHead, _ := excelize.CoordinatesToCellName(len(heads), 1)
	f.SetCellStyle(sheet, "A1", lastHead, headerStyle)
	f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	for i, e := range rows {
		row := i + 2
		set := func(col int, v any) {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellValue(sheet, cell, v)
		}
		set(1, e.number)
		set(2, e.name)
		set(3, e.department)
		for j, c := range totalsColumns {
			set(4+j, c.value(e.totals))
		}
	}
}

type employeeTotals struct {
	number, name, department string
	totals                   Totals
}

func writeSummarySheet(f *excelize.File, r *Monthly, headerStyle int) {
	rows := make([]employeeTotals, 0, len(r.Employees))
	for _, e := range r.Employees {
		rows = append(rows, employeeTotals{e.EmployeeNumber, e.Name, e.DepartmentName, e.Totals})
	}
	writeTotalsTable(f, sheetSummary, headerStyle, rows)
}

// gridTotalHeads label writeDayGrid's optional per-month count columns,
// using the same codes as the grid itself.
var gridTotalHeads = []string{"H", "T", "I", "S", "C", "A"}

func writeDetailSheet(f *excelize.File, r *Monthly, headerStyle int, fl fills) {
	writeDayGrid(f, sheetDetail, r, headerStyle, fl, false)
}

// writeDayGrid writes the NIK/Nama + one-column-per-day code grid for r to
// sheet. withTotals appends the month's H/T/I/S/C/A counts after the last
// day, so a yearly workbook's month sheet can be checked on its own.
func writeDayGrid(f *excelize.File, sheet string, r *Monthly, headerStyle int, fl fills, withTotals bool) {
	f.SetColWidth(sheet, "A", "A", 14)
	f.SetColWidth(sheet, "B", "B", 28)

	f.SetCellValue(sheet, "A1", "NIK")
	f.SetCellValue(sheet, "B1", "Nama")
	for day := 1; day <= r.DaysInMonth; day++ {
		col := day + 2
		colName, _ := excelize.ColumnNumberToName(col)
		f.SetColWidth(sheet, colName, colName, 4.5)
		cell, _ := excelize.CoordinatesToCellName(col, 1)
		f.SetCellValue(sheet, cell, day)
	}
	lastColNum := r.DaysInMonth + 2
	if withTotals {
		for i, h := range gridTotalHeads {
			col := r.DaysInMonth + 3 + i
			colName, _ := excelize.ColumnNumberToName(col)
			f.SetColWidth(sheet, colName, colName, 6)
			cell, _ := excelize.CoordinatesToCellName(col, 1)
			f.SetCellValue(sheet, cell, h)
		}
		lastColNum += len(gridTotalHeads)
	}
	lastCol, _ := excelize.ColumnNumberToName(lastColNum)
	f.SetCellStyle(sheet, "A1", lastCol+"1", headerStyle)
	f.SetPanes(sheet, &excelize.Panes{Freeze: true, XSplit: 2, YSplit: 1, TopLeftCell: "C2", ActivePane: "bottomRight"})

	for i, e := range r.Employees {
		row := i + 2
		a1, _ := excelize.CoordinatesToCellName(1, row)
		b1, _ := excelize.CoordinatesToCellName(2, row)
		f.SetCellValue(sheet, a1, e.EmployeeNumber)
		f.SetCellValue(sheet, b1, e.Name)

		for _, cell := range e.Days {
			col := cell.Day + 2
			ref, _ := excelize.CoordinatesToCellName(col, row)
			text, style := detailCell(cell, fl)
			if text != "" {
				f.SetCellValue(sheet, ref, text)
			}
			if style != 0 {
				f.SetCellStyle(sheet, ref, ref, style)
			}
		}

		if withTotals {
			t := e.Totals
			for i, v := range []int{t.OnTime, t.LateCount, t.Izin, t.Sakit, t.Cuti, t.Absent} {
				ref, _ := excelize.CoordinatesToCellName(r.DaysInMonth+3+i, row)
				f.SetCellValue(sheet, ref, v)
			}
		}
	}
}

func detailCell(c DayCell, fl fills) (text string, style int) {
	switch c.Status {
	case DayOnTime:
		return "H", 0
	case DayLate:
		return "T" + strconv.Itoa(c.LateMinutes), fl.late
	case DayIzin:
		return "I", fl.izin
	case DaySakit:
		return "S", fl.sakit
	case DayCuti:
		return "C", fl.cuti
	case DayAbsent:
		return "A", fl.absent
	case DayOff:
		return "", fl.off
	default: // DayPending
		return "", 0
	}
}

func writeLegendSheet(f *excelize.File, r *Monthly, headerStyle int) {
	writeLegend(f, headerStyle, [][2]string{
		{"Laporan", fmt.Sprintf("Kehadiran %s %d", monthNamesID[r.Month], r.Year)},
		{"Dibuat", r.GeneratedAt.Format("2 January 2006 15:04") + " WIB"},
		{"Jumlah karyawan", strconv.Itoa(len(r.Employees))},
	}, "Arti (sheet Detail Harian)")
}

// writeLegend writes the info rows, a blank row, then the day-code legend
// (headed codesHead) to the Keterangan sheet.
func writeLegend(f *excelize.File, headerStyle int, info [][2]string, codesHead string) {
	f.SetColWidth(sheetLegend, "A", "A", 16)
	f.SetColWidth(sheetLegend, "B", "B", 60)

	rows := append(info, [][2]string{
		{"", ""},
		{"Kode", codesHead},
		{"H", "Hadir tepat waktu"},
		{"T<n>", "Terlambat <n> menit (mis. T15 = telat 15 menit)"},
		{"I", "Izin"},
		{"S", "Sakit"},
		{"C", "Cuti"},
		{"A", "Tidak hadir tanpa keterangan (alpha)"},
		{"(kosong)", "Bukan hari kerja, atau tanggal belum berjalan"},
	}...)
	for i, kv := range rows {
		ra, _ := excelize.CoordinatesToCellName(1, i+1)
		rb, _ := excelize.CoordinatesToCellName(2, i+1)
		f.SetCellValue(sheetLegend, ra, kv[0])
		f.SetCellValue(sheetLegend, rb, kv[1])
	}
	headRow := strconv.Itoa(len(info) + 2)
	f.SetCellStyle(sheetLegend, "A"+headRow, "B"+headRow, headerStyle)
}

// FileName is the download filename for a month's report.
func FileName(year, month int) string {
	return fmt.Sprintf("laporan-kehadiran-%04d-%02d.xlsx", year, month)
}
