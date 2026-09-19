package leave

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/middleware"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
	"github.com/suryaintigas/absensi-backend/pkg/rbac"
	"github.com/suryaintigas/absensi-backend/pkg/response"
	"github.com/suryaintigas/absensi-backend/pkg/validator"
)

const dateLayout = "2006-01-02"

type RequestBody struct {
	EmployeeID uuid.UUID `json:"employee_id" validate:"required"`
	StartDate  string    `json:"start_date" validate:"required"`
	EndDate    string    `json:"end_date" validate:"required"`
	Reason     string    `json:"reason" validate:"omitempty,max=255"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// actor resolves the audit-trail Actor for the currently authenticated
// request straight from the JWT claims middleware.AuthRequired already put
// in context — see employee.Handler.actor's doc comment.
func (h *Handler) actor(c *gin.Context) Actor {
	id, _ := uuid.Parse(c.GetString(middleware.ContextKeyUserID))
	role := rbac.Role(c.GetString(middleware.ContextKeyUserRole))
	name := c.GetString(middleware.ContextKeyUserName)
	return Actor{ID: id, Name: name, Role: role, IP: c.ClientIP()}
}

func (h *Handler) Create(c *gin.Context) {
	var req RequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	start, end, err := parseRange(req.StartDate, req.EndDate)
	if err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}

	l, err := h.service.Request(c.Request.Context(), h.actor(c), Input{
		EmployeeID: req.EmployeeID,
		StartDate:  start,
		EndDate:    end,
		Reason:     req.Reason,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusCreated, "Cuti berhasil dicatat", toData(l))
}

func (h *Handler) List(c *gin.Context) {
	q := c.Request.URL.Query()
	p := pagination.FromQuery(q)

	f := Filter{Status: q.Get("status")}
	if raw := q.Get("employee_id"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			f.EmployeeID = &id
		}
	}

	items, total, err := h.service.List(c.Request.Context(), f, p)
	if err != nil {
		slog.Error("leave_list_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil data cuti", nil)
		return
	}

	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, toData(&items[i]))
	}
	response.OK(c, http.StatusOK, "Daftar cuti", gin.H{
		"items": list,
		"meta":  pagination.NewMeta(p, total),
	})
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	l, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Detail cuti", toData(l))
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	if err := h.service.Cancel(c.Request.Context(), h.actor(c), id); err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Cuti berhasil dibatalkan", nil)
}

// Balance reports an employee's annual leave usage for a given year
// (defaulting to the current year).
func (h *Handler) Balance(c *gin.Context) {
	employeeID, err := uuid.Parse(c.Query("employee_id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "employee_id tidak valid", nil)
		return
	}
	year := time.Now().Year()
	if raw := c.Query("year"); raw != "" {
		if parsed, err := time.Parse("2006", raw); err == nil {
			year = parsed.Year()
		}
	}

	used, remaining, err := h.service.Balance(c.Request.Context(), employeeID, year)
	if err != nil {
		slog.Error("leave_balance_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil sisa cuti", nil)
		return
	}
	response.OK(c, http.StatusOK, "Sisa cuti", gin.H{
		"employee_id": employeeID,
		"year":        year,
		"quota":       AnnualQuotaDays,
		"used":        used,
		"remaining":   remaining,
	})
}

func parseRange(startRaw, endRaw string) (start, end time.Time, err error) {
	start, err = time.Parse(dateLayout, startRaw)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("start_date harus berformat YYYY-MM-DD")
	}
	end, err = time.Parse(dateLayout, endRaw)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("end_date harus berformat YYYY-MM-DD")
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, errors.New("end_date tidak boleh sebelum start_date")
	}
	return start, end, nil
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "Cuti tidak ditemukan", nil)
	case errors.Is(err, ErrInvalidEmployee):
		response.Fail(c, http.StatusUnprocessableEntity, "Karyawan tidak ditemukan", nil)
	case errors.Is(err, ErrAlreadyCancelled):
		response.Fail(c, http.StatusConflict, "Cuti sudah dibatalkan", nil)
	case errors.Is(err, ErrQuotaExceeded):
		response.Fail(c, http.StatusUnprocessableEntity, "Sisa jatah cuti tahunan tidak mencukupi (maksimal 12 hari/tahun)", nil)
	default:
		slog.Error("leave_unhandled_error", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Terjadi kesalahan. Silakan coba lagi.", nil)
	}
}

func toData(l *Leave) gin.H {
	return gin.H{
		"id":              l.ID,
		"employee_id":     l.EmployeeID,
		"employee_name":   l.EmployeeName,
		"employee_number": l.EmployeeNumber,
		"start_date":      l.StartDate.Format(dateLayout),
		"end_date":        l.EndDate.Format(dateLayout),
		"days_count":      l.DaysCount,
		"reason":          l.Reason,
		"status":          l.Status,
		"created_at":      l.CreatedAt,
		"updated_at":      l.UpdatedAt,
	}
}
