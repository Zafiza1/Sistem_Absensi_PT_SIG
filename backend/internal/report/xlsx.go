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
	lateFill, _ := f.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FEF3C7"}}})
	absentFill, _ := f.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FEE2E2"}}})
	offFill, _ := f.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F1F5F9"}}})

	writeSummarySheet(f, r, header)
	writeDetailSheet(f, r, header, lateFill, absentFill, offFill)
	writeLegendSheet(f, r, header)

	f.SetActiveSheet(0)
	return f.WriteToBuffer()
}

func writeSummarySheet(f *excelize.File, r *Monthly, headerStyle int) {
	cols := []struct {
		head  string
		width float64
	}{
		{"NIK", 14}, {"Nama", 28}, {"Divisi", 20},
		{"Hari Kerja", 11}, {"Tepat Waktu", 12}, {"Terlambat (kali)", 15},
		{"Total Menit Terlambat", 20}, {"Tidak Hadir", 12},
	}
	for i, c := range cols {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetSummary, col, col, c.width)
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetSummary, cell, c.head)
	}
	f.SetCellStyle(sheetSummary, "A1", "H1", headerStyle)
	f.SetPanes(sheetSummary, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	for i, e := range r.Employees {
		row := i + 2
		set := func(col int, v any) {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellValue(sheetSummary, cell, v)
		}
		set(1, e.EmployeeNumber)
		set(2, e.Name)
		set(3, e.DepartmentName)
		set(4, e.WorkingDays)
		set(5, e.OnTime)
		set(6, e.LateCount)
		set(7, e.LateMinutes)
		set(8, e.Absent)
	}
}

func writeDetailSheet(f *excelize.File, r *Monthly, headerStyle, lateFill, absentFill, offFill int) {
	f.SetColWidth(sheetDetail, "A", "A", 14)
	f.SetColWidth(sheetDetail, "B", "B", 28)

	f.SetCellValue(sheetDetail, "A1", "NIK")
	f.SetCellValue(sheetDetail, "B1", "Nama")
	for day := 1; day <= r.DaysInMonth; day++ {
		col := day + 2
		colName, _ := excelize.ColumnNumberToName(col)
		f.SetColWidth(sheetDetail, colName, colName, 4.5)
		cell, _ := excelize.CoordinatesToCellName(col, 1)
		f.SetCellValue(sheetDetail, cell, day)
	}
	lastCol, _ := excelize.ColumnNumberToName(r.DaysInMonth + 2)
	f.SetCellStyle(sheetDetail, "A1", lastCol+"1", headerStyle)
	f.SetPanes(sheetDetail, &excelize.Panes{Freeze: true, XSplit: 2, YSplit: 1, TopLeftCell: "C2", ActivePane: "bottomRight"})

	for i, e := range r.Employees {
		row := i + 2
		a1, _ := excelize.CoordinatesToCellName(1, row)
		b1, _ := excelize.CoordinatesToCellName(2, row)
		f.SetCellValue(sheetDetail, a1, e.EmployeeNumber)
		f.SetCellValue(sheetDetail, b1, e.Name)

		for _, cell := range e.Days {
			col := cell.Day + 2
			ref, _ := excelize.CoordinatesToCellName(col, row)
			text, style := detailCell(cell, lateFill, absentFill, offFill)
			if text != "" {
				f.SetCellValue(sheetDetail, ref, text)
			}
			if style != 0 {
				f.SetCellStyle(sheetDetail, ref, ref, style)
			}
		}
	}
}

func detailCell(c DayCell, lateFill, absentFill, offFill int) (text string, style int) {
	switch c.Status {
	case DayOnTime:
		return "H", 0
	case DayLate:
		return "T" + strconv.Itoa(c.LateMinutes), lateFill
	case DayAbsent:
		return "A", absentFill
	case DayOff:
		return "", offFill
	default: // DayPending
		return "", 0
	}
}

func writeLegendSheet(f *excelize.File, r *Monthly, headerStyle int) {
	f.SetColWidth(sheetLegend, "A", "A", 16)
	f.SetColWidth(sheetLegend, "B", "B", 60)

	rows := [][2]string{
		{"Laporan", fmt.Sprintf("Kehadiran %s %d", monthNamesID[r.Month], r.Year)},
		{"Dibuat", r.GeneratedAt.Format("2 January 2006 15:04") + " WIB"},
		{"Jumlah karyawan", strconv.Itoa(len(r.Employees))},
		{"", ""},
		{"Kode", "Arti (sheet Detail Harian)"},
		{"H", "Hadir tepat waktu"},
		{"T<n>", "Terlambat <n> menit (mis. T15 = telat 15 menit)"},
		{"A", "Tidak hadir pada hari kerja"},
		{"(kosong)", "Bukan hari kerja, atau tanggal belum berjalan"},
	}
	for i, kv := range rows {
		ra, _ := excelize.CoordinatesToCellName(1, i+1)
		rb, _ := excelize.CoordinatesToCellName(2, i+1)
		f.SetCellValue(sheetLegend, ra, kv[0])
		f.SetCellValue(sheetLegend, rb, kv[1])
	}
	f.SetCellStyle(sheetLegend, "A5", "B5", headerStyle)
}

// FileName is the download filename for a month's report.
func FileName(year, month int) string {
	return fmt.Sprintf("laporan-kehadiran-%04d-%02d.xlsx", year, month)
}
