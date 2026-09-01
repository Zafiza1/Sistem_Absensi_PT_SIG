package report

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/response"
)

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Monthly serves GET /reports/monthly?month=YYYY-MM[&department_id=][&format=xlsx].
// Without format=xlsx it returns the report as JSON; with it, an .xlsx
// download.
func (h *Handler) Monthly(c *gin.Context) {
	raw := c.Query("month")
	m, err := time.Parse("2006-01", raw)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Parameter month harus format YYYY-MM", nil)
		return
	}
	year, month := m.Year(), int(m.Month())

	var departmentID *uuid.UUID
	if v := c.Query("department_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "department_id tidak valid", nil)
			return
		}
		departmentID = &id
	}

	rpt, err := h.service.Monthly(c.Request.Context(), year, month, departmentID)
	if err != nil {
		if errors.Is(err, ErrInvalidMonth) {
			response.Fail(c, http.StatusBadRequest, "Bulan tidak valid", nil)
			return
		}
		slog.Error("report_monthly_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal menyusun laporan", nil)
		return
	}

	if c.Query("format") == "xlsx" {
		buf, err := BuildXLSX(rpt)
		if err != nil {
			slog.Error("report_monthly_xlsx_failed", slog.String("error", err.Error()))
			response.Fail(c, http.StatusInternalServerError, "Gagal membuat file Excel", nil)
			return
		}
		c.Header("Content-Disposition", `attachment; filename="`+FileName(year, month)+`"`)
		c.Data(http.StatusOK, xlsxContentType, buf.Bytes())
		return
	}

	response.OK(c, http.StatusOK, "Laporan kehadiran bulanan", toData(rpt))
}

func toData(r *Monthly) gin.H {
	employees := make([]gin.H, 0, len(r.Employees))
	for _, e := range r.Employees {
		days := make([]gin.H, 0, len(e.Days))
		for _, d := range e.Days {
			days = append(days, gin.H{
				"day":          d.Day,
				"status":       d.Status,
				"late_minutes": d.LateMinutes,
				"check_in_at":  d.CheckInAt,
				"check_out_at": d.CheckOutAt,
			})
		}
		employees = append(employees, gin.H{
			"employee_id":     e.EmployeeID,
			"employee_number": e.EmployeeNumber,
			"name":            e.Name,
			"department_name": e.DepartmentName,
			"working_days":    e.WorkingDays,
			"on_time":         e.OnTime,
			"late_count":      e.LateCount,
			"late_minutes":    e.LateMinutes,
			"absent":          e.Absent,
			"days":            days,
		})
	}
	return gin.H{
		"year":          r.Year,
		"month":         r.Month,
		"days_in_month": r.DaysInMonth,
		"generated_at":  r.GeneratedAt,
		"employees":     employees,
	}
}
