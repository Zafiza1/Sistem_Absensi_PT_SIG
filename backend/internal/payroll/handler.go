package payroll

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Monthly handles GET /payroll/monthly?year=&month=&department_id=,
// defaulting year/month to the current Jakarta calendar month.
func (h *Handler) Monthly(c *gin.Context) {
	q := c.Request.URL.Query()

	now := time.Now().In(jakarta)
	year := now.Year()
	if raw := q.Get("year"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			year = parsed
		}
	}
	month := int(now.Month())
	if raw := q.Get("month"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			month = parsed
		}
	}

	var departmentID *uuid.UUID
	if raw := q.Get("department_id"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			departmentID = &id
		}
	}

	report, err := h.service.Monthly(c.Request.Context(), year, month, departmentID)
	if err != nil {
		if errors.Is(err, ErrInvalidMonth) {
			response.Fail(c, http.StatusUnprocessableEntity, "Bulan harus antara 1 dan 12", nil)
			return
		}
		slog.Error("payroll_monthly_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal membuat laporan payroll", nil)
		return
	}

	items := make([]gin.H, 0, len(report.Employees))
	for _, ep := range report.Employees {
		items = append(items, gin.H{
			"employee_id":          ep.EmployeeID,
			"employee_number":      ep.EmployeeNumber,
			"name":                 ep.Name,
			"department_name":      ep.DepartmentName,
			"base_salary":          ep.BaseSalary,
			"late_days":            ep.LateDays,
			"late_deduction":       ep.LateDeduction,
			"net_salary":           ep.NetSalary,
			"leave_days_used":      ep.LeaveDaysUsed,
			"leave_days_remaining": ep.LeaveDaysRemaining,
		})
	}
	response.OK(c, http.StatusOK, "Laporan payroll bulanan", gin.H{
		"year":         report.Year,
		"month":        report.Month,
		"generated_at": report.GeneratedAt,
		"employees":    items,
	})
}
