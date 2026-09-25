package payroll

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/middleware"
	"github.com/suryaintigas/absensi-backend/pkg/response"
	"github.com/suryaintigas/absensi-backend/pkg/validator"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// actor resolves the audit-trail Actor for the currently authenticated
// request straight from the JWT claims middleware.AuthRequired already put
// in context.
func (h *Handler) actor(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(c.GetString(middleware.ContextKeyUserID))
	return id
}

// Monthly handles GET /payroll/monthly?year=&month=&department_id=,
// defaulting year/month to the current Jakarta calendar month (legacy compatibility).
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

// Payroll Period Management

type CreatePeriodRequest struct {
	Year  int    `json:"year" validate:"required,min=2020,max=2100"`
	Month int    `json:"month" validate:"required,min=1,max=12"`
	Notes string `json:"notes" validate:"max=500"`
}

// CreatePayrollPeriod handles POST /payroll/periods
func (h *Handler) CreatePayrollPeriod(c *gin.Context) {
	var req CreatePeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	period, err := h.service.CreatePayrollPeriod(c.Request.Context(), req.Year, req.Month, req.Notes)
	if err != nil {
		if errors.Is(err, ErrPeriodExists) {
			response.Fail(c, http.StatusConflict, "Periode payroll untuk bulan ini sudah ada", nil)
			return
		}
		if errors.Is(err, ErrInvalidMonth) {
			response.Fail(c, http.StatusUnprocessableEntity, "Bulan harus antara 1 dan 12", nil)
			return
		}
		slog.Error("create_payroll_period_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal membuat periode payroll", nil)
		return
	}

	response.OK(c, http.StatusCreated, "Periode payroll berhasil dibuat", toPeriodData(period))
}

// ListPayrollPeriods handles GET /payroll/periods?year=&status=
func (h *Handler) ListPayrollPeriods(c *gin.Context) {
	q := c.Request.URL.Query()

	var year *int
	if raw := q.Get("year"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			year = &parsed
		}
	}

	var status *string
	if raw := q.Get("status"); raw != "" {
		status = &raw
	}

	periods, err := h.service.ListPayrollPeriods(c.Request.Context(), year, status)
	if err != nil {
		slog.Error("list_payroll_periods_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil daftar periode payroll", nil)
		return
	}

	items := make([]gin.H, 0, len(periods))
	for _, p := range periods {
		items = append(items, toPeriodData(&p))
	}

	response.OK(c, http.StatusOK, "Daftar periode payroll", gin.H{
		"items": items,
	})
}

// GetPayrollPeriod handles GET /payroll/periods/:id
func (h *Handler) GetPayrollPeriod(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	period, err := h.service.GetPayrollPeriod(c.Request.Context(), id)
	if err != nil {
		slog.Error("get_payroll_period_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusNotFound, "Periode payroll tidak ditemukan", nil)
		return
	}

	response.OK(c, http.StatusOK, "Detail periode payroll", toPeriodData(period))
}

// ProcessPayrollPeriod handles POST /payroll/periods/:id/process
func (h *Handler) ProcessPayrollPeriod(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	period, err := h.service.ProcessPayrollPeriod(c.Request.Context(), id, h.actor(c))
	if err != nil {
		if errors.Is(err, ErrPeriodLocked) {
			response.Fail(c, http.StatusForbidden, "Periode payroll terkunci dan tidak dapat dimodifikasi", nil)
			return
		}
		if errors.Is(err, ErrInvalidStatus) {
			response.Fail(c, http.StatusBadRequest, "Hanya periode draft dapat diproses", nil)
			return
		}
		slog.Error("process_payroll_period_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal memproses payroll", nil)
		return
	}

	response.OK(c, http.StatusOK, "Payroll berhasil diproses", toPeriodData(period))
}

// LockPayrollPeriod handles POST /payroll/periods/:id/lock
func (h *Handler) LockPayrollPeriod(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	if err := h.service.LockPayrollPeriod(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrInvalidStatus) {
			response.Fail(c, http.StatusBadRequest, "Hanya periode yang sudah selesai dapat dikunci", nil)
			return
		}
		slog.Error("lock_payroll_period_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengunci periode payroll", nil)
		return
	}

	response.OK(c, http.StatusOK, "Periode payroll berhasil dikunci", nil)
}

// DeletePayrollPeriod handles DELETE /payroll/periods/:id
func (h *Handler) DeletePayrollPeriod(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	if err := h.service.DeletePayrollPeriod(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrInvalidStatus) {
			response.Fail(c, http.StatusBadRequest, "Hanya periode draft dapat dihapus", nil)
			return
		}
		slog.Error("delete_payroll_period_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal menghapus periode payroll", nil)
		return
	}

	response.OK(c, http.StatusOK, "Periode payroll berhasil dihapus", nil)
}

// Payroll Item Management

// GetPayrollItems handles GET /payroll/periods/:id/items
func (h *Handler) GetPayrollItems(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	items, err := h.service.GetPayrollItemsByPeriod(c.Request.Context(), id)
	if err != nil {
		slog.Error("get_payroll_items_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil item payroll", nil)
		return
	}

	data := make([]gin.H, 0, len(items))
	for _, item := range items {
		data = append(data, toPayrollItemData(&item))
	}

	response.OK(c, http.StatusOK, "Item payroll", gin.H{
		"items": data,
	})
}

type UpdatePayrollItemRequest struct {
	Allowance          *int64 `json:"allowance" validate:"omitempty,min=0"`
	Bonus              *int64 `json:"bonus" validate:"omitempty,min=0"`
	OtherEarnings      *int64 `json:"other_earnings" validate:"omitempty,min=0"`
	TaxDeduction       *int64 `json:"tax_deduction" validate:"omitempty,min=0"`
	InsuranceDeduction *int64 `json:"insurance_deduction" validate:"omitempty,min=0"`
	OtherDeductions    *int64 `json:"other_deductions" validate:"omitempty,min=0"`
	Notes              string `json:"notes" validate:"max=500"`
}

// UpdatePayrollItem handles PUT /payroll/items/:id
func (h *Handler) UpdatePayrollItem(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	var req UpdatePayrollItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	item, err := h.service.GetPayrollItem(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "Item payroll tidak ditemukan", nil)
		return
	}

	if period, perr := h.service.GetPayrollPeriod(c.Request.Context(), item.PayrollPeriodID); perr == nil && period.Status == "LOCKED" {
		response.Fail(c, http.StatusForbidden, "Periode payroll terkunci dan tidak dapat dimodifikasi", nil)
		return
	}

	if req.Allowance != nil {
		item.Allowance = *req.Allowance
	}
	if req.Bonus != nil {
		item.Bonus = *req.Bonus
	}
	if req.OtherEarnings != nil {
		item.OtherEarnings = *req.OtherEarnings
	}
	if req.TaxDeduction != nil {
		item.TaxDeduction = *req.TaxDeduction
	}
	if req.InsuranceDeduction != nil {
		item.InsuranceDeduction = *req.InsuranceDeduction
	}
	if req.OtherDeductions != nil {
		item.OtherDeductions = *req.OtherDeductions
	}
	item.Notes = req.Notes

	if err := h.service.UpdatePayrollItem(c.Request.Context(), item); err != nil {
		slog.Error("update_payroll_item_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal memperbarui item payroll", nil)
		return
	}

	response.OK(c, http.StatusOK, "Item payroll berhasil diperbarui", toPayrollItemData(item))
}

type MarkAsPaidRequest struct {
	PaymentDate      string `json:"payment_date" validate:"required"`
	PaymentMethod    string `json:"payment_method" validate:"required,max=50"`
	PaymentReference string `json:"payment_reference" validate:"max=100"`
}

// MarkAsPaid handles POST /payroll/periods/:id/mark-paid
func (h *Handler) MarkAsPaid(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	var req MarkAsPaidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	paymentDate, err := time.Parse("2006-01-02", req.PaymentDate)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Format tanggal tidak valid (YYYY-MM-DD)", nil)
		return
	}

	if err := h.service.MarkAsPaid(c.Request.Context(), id, paymentDate, req.PaymentMethod, req.PaymentReference); err != nil {
		slog.Error("mark_as_paid_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal menandai sebagai dibayar", nil)
		return
	}

	response.OK(c, http.StatusOK, "Payroll berhasil ditandai sebagai dibayar", nil)
}

// Deduction Rule Management

type CreateDeductionRuleRequest struct {
	RuleType        string   `json:"rule_type" validate:"required,oneof=LATE ABSENT OTHER"`
	RuleName        string   `json:"rule_name" validate:"required,min=2,max=100"`
	Description     string   `json:"description" validate:"max=500"`
	LateMinMinutes  *int     `json:"late_min_minutes" validate:"omitempty,min=0"`
	LateMaxMinutes  *int     `json:"late_max_minutes" validate:"omitempty,min=0"`
	DeductionAmount int64    `json:"deduction_amount" validate:"required,min=0"`
	DeductionType   string   `json:"deduction_type" validate:"required,oneof=FIXED PERCENTAGE HALF_DAY_SALARY"`
	PercentageValue *float64 `json:"percentage_value" validate:"omitempty,min=0,max=100"`
	Priority        int      `json:"priority" validate:"omitempty,min=0"`
}

// CreateDeductionRule handles POST /payroll/deduction-rules
func (h *Handler) CreateDeductionRule(c *gin.Context) {
	var req CreateDeductionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	rule := &DeductionRule{
		ID:              uuid.New(),
		RuleType:        req.RuleType,
		RuleName:        req.RuleName,
		Description:     req.Description,
		LateMinMinutes:  req.LateMinMinutes,
		LateMaxMinutes:  req.LateMaxMinutes,
		DeductionAmount: req.DeductionAmount,
		DeductionType:   req.DeductionType,
		PercentageValue: req.PercentageValue,
		IsActive:        true,
		EffectiveDate:   time.Now().In(jakarta),
		Priority:        req.Priority,
	}

	if err := h.service.CreateDeductionRule(c.Request.Context(), rule); err != nil {
		slog.Error("create_deduction_rule_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal membuat aturan potongan", nil)
		return
	}

	response.OK(c, http.StatusCreated, "Aturan potongan berhasil dibuat", toDeductionRuleData(rule))
}

// ListDeductionRules handles GET /payroll/deduction-rules?rule_type=&is_active=
func (h *Handler) ListDeductionRules(c *gin.Context) {
	q := c.Request.URL.Query()

	var ruleType *string
	if raw := q.Get("rule_type"); raw != "" {
		ruleType = &raw
	}

	var isActive *bool
	if raw := q.Get("is_active"); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			isActive = &parsed
		}
	}

	rules, err := h.service.ListDeductionRules(c.Request.Context(), ruleType, isActive)
	if err != nil {
		slog.Error("list_deduction_rules_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil daftar aturan potongan", nil)
		return
	}

	items := make([]gin.H, 0, len(rules))
	for _, rule := range rules {
		items = append(items, toDeductionRuleData(&rule))
	}

	response.OK(c, http.StatusOK, "Daftar aturan potongan", gin.H{
		"items": items,
	})
}

type UpdateDeductionRuleRequest struct {
	RuleType        string   `json:"rule_type" validate:"required,oneof=LATE ABSENT OTHER"`
	RuleName        string   `json:"rule_name" validate:"required,min=2,max=100"`
	Description     string   `json:"description" validate:"max=500"`
	LateMinMinutes  *int     `json:"late_min_minutes" validate:"omitempty,min=0"`
	LateMaxMinutes  *int     `json:"late_max_minutes" validate:"omitempty,min=0"`
	DeductionAmount int64    `json:"deduction_amount" validate:"required,min=0"`
	DeductionType   string   `json:"deduction_type" validate:"required,oneof=FIXED PERCENTAGE HALF_DAY_SALARY"`
	PercentageValue *float64 `json:"percentage_value" validate:"omitempty,min=0,max=100"`
	IsActive        bool     `json:"is_active"`
	Priority        int      `json:"priority" validate:"omitempty,min=0"`
}

// UpdateDeductionRule handles PUT /payroll/deduction-rules/:id
func (h *Handler) UpdateDeductionRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	var req UpdateDeductionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	rule, err := h.service.GetDeductionRule(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "Aturan potongan tidak ditemukan", nil)
		return
	}

	rule.RuleType = req.RuleType
	rule.RuleName = req.RuleName
	rule.Description = req.Description
	rule.LateMinMinutes = req.LateMinMinutes
	rule.LateMaxMinutes = req.LateMaxMinutes
	rule.DeductionAmount = req.DeductionAmount
	rule.DeductionType = req.DeductionType
	rule.PercentageValue = req.PercentageValue
	rule.IsActive = req.IsActive
	rule.Priority = req.Priority

	if err := h.service.UpdateDeductionRule(c.Request.Context(), rule); err != nil {
		slog.Error("update_deduction_rule_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal memperbarui aturan potongan", nil)
		return
	}

	response.OK(c, http.StatusOK, "Aturan potongan berhasil diperbarui", toDeductionRuleData(rule))
}

// DeleteDeductionRule handles DELETE /payroll/deduction-rules/:id
func (h *Handler) DeleteDeductionRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	if err := h.service.DeleteDeductionRule(c.Request.Context(), id); err != nil {
		slog.Error("delete_deduction_rule_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal menghapus aturan potongan", nil)
		return
	}

	response.OK(c, http.StatusOK, "Aturan potongan berhasil dihapus", nil)
}

// Helper functions

func toPeriodData(p *PayrollPeriod) gin.H {
	return gin.H{
		"id":               p.ID,
		"period_start":     p.PeriodStart,
		"period_end":       p.PeriodEnd,
		"year":             p.Year,
		"month":            p.Month,
		"status":           p.Status,
		"processed_at":     p.ProcessedAt,
		"processed_by":     p.ProcessedBy,
		"total_employees":  p.TotalEmployees,
		"total_gross_pay":  p.TotalGrossPay,
		"total_net_pay":    p.TotalNetPay,
		"total_deductions": p.TotalDeductions,
		"notes":            p.Notes,
		"created_at":       p.CreatedAt,
		"updated_at":       p.UpdatedAt,
	}
}

func toPayrollItemData(item *PayrollItem) gin.H {
	return gin.H{
		"id":                  item.ID,
		"payroll_period_id":   item.PayrollPeriodID,
		"employee_id":         item.EmployeeID,
		"employee_number":     item.EmployeeNumber,
		"employee_name":       item.EmployeeName,
		"department_id":       item.DepartmentID,
		"department_name":     item.DepartmentName,
		"position_id":         item.PositionID,
		"position_name":       item.PositionName,
		"working_days":        item.WorkingDays,
		"present_days":        item.PresentDays,
		"absent_days":         item.AbsentDays,
		"late_days":           item.LateDays,
		"late_minutes":        item.LateMinutes,
		"leave_days":          item.LeaveDays,
		"base_salary":         item.BaseSalary,
		"overtime_hours":      item.OvertimeHours,
		"overtime_pay":        item.OvertimePay,
		"allowance":           item.Allowance,
		"bonus":               item.Bonus,
		"other_earnings":      item.OtherEarnings,
		"total_earnings":      item.TotalEarnings,
		"late_deduction":      item.LateDeduction,
		"absent_deduction":    item.AbsentDeduction,
		"tax_deduction":       item.TaxDeduction,
		"insurance_deduction": item.InsuranceDeduction,
		"other_deductions":    item.OtherDeductions,
		"total_deductions":    item.TotalDeductions,
		"gross_pay":           item.GrossPay,
		"net_pay":             item.NetPay,
		"payment_status":      item.PaymentStatus,
		"payment_date":        item.PaymentDate,
		"payment_method":      item.PaymentMethod,
		"payment_reference":   item.PaymentReference,
		"notes":               item.Notes,
		"created_at":          item.CreatedAt,
		"updated_at":          item.UpdatedAt,
	}
}

func toDeductionRuleData(rule *DeductionRule) gin.H {
	return gin.H{
		"id":               rule.ID,
		"rule_type":        rule.RuleType,
		"rule_name":        rule.RuleName,
		"description":      rule.Description,
		"late_min_minutes": rule.LateMinMinutes,
		"late_max_minutes": rule.LateMaxMinutes,
		"deduction_amount": rule.DeductionAmount,
		"deduction_type":   rule.DeductionType,
		"percentage_value": rule.PercentageValue,
		"is_active":        rule.IsActive,
		"effective_date":   rule.EffectiveDate,
		"expiry_date":      rule.ExpiryDate,
		"priority":         rule.Priority,
		"created_at":       rule.CreatedAt,
		"updated_at":       rule.UpdatedAt,
	}
}
