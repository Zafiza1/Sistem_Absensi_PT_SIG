package schedule

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/pagination"
	"github.com/suryaintigas/absensi-backend/pkg/response"
	"github.com/suryaintigas/absensi-backend/pkg/validator"
)

type CreateRequest struct {
	EmployeeID uuid.UUID `json:"employee_id" validate:"required"`
	ShiftID    uuid.UUID `json:"shift_id" validate:"required"`
	DayOfWeek  int       `json:"day_of_week" validate:"required,gte=1,lte=7"`
}

type UpdateRequest struct {
	ShiftID   uuid.UUID `json:"shift_id" validate:"required"`
	DayOfWeek int       `json:"day_of_week" validate:"required,gte=1,lte=7"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	sc, err := h.service.Create(c.Request.Context(), req.EmployeeID, req.ShiftID, req.DayOfWeek)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusCreated, "Jadwal berhasil dibuat", toData(sc))
}

func (h *Handler) List(c *gin.Context) {
	q := c.Request.URL.Query()

	if raw := q.Get("employee_id"); raw != "" {
		employeeID, err := uuid.Parse(raw)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "employee_id tidak valid", nil)
			return
		}
		items, err := h.service.ListByEmployee(c.Request.Context(), employeeID)
		if err != nil {
			slog.Error("schedule_list_by_employee_failed", slog.String("error", err.Error()))
			response.Fail(c, http.StatusInternalServerError, "Gagal mengambil data jadwal", nil)
			return
		}
		list := make([]gin.H, 0, len(items))
		for i := range items {
			list = append(list, toData(&items[i]))
		}
		response.OK(c, http.StatusOK, "Jadwal karyawan", gin.H{"items": list})
		return
	}

	p := pagination.FromQuery(q)
	items, total, err := h.service.List(c.Request.Context(), p)
	if err != nil {
		slog.Error("schedule_list_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil data jadwal", nil)
		return
	}
	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, toData(&items[i]))
	}
	response.OK(c, http.StatusOK, "Daftar jadwal", gin.H{
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

	sc, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Detail jadwal", toData(sc))
}

func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	sc, err := h.service.Update(c.Request.Context(), id, req.ShiftID, req.DayOfWeek)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Jadwal berhasil diperbarui", toData(sc))
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Jadwal berhasil dihapus", nil)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "Jadwal tidak ditemukan", nil)
	case errors.Is(err, ErrAlreadyAssigned):
		response.Fail(c, http.StatusConflict, "Karyawan sudah memiliki shift untuk hari ini", nil)
	case errors.Is(err, ErrInvalidReference):
		response.Fail(c, http.StatusUnprocessableEntity, "Karyawan atau shift tidak ditemukan", nil)
	case errors.Is(err, ErrInvalidDayOfWeek):
		response.Fail(c, http.StatusUnprocessableEntity, "Hari harus antara 1 (Senin) - 7 (Minggu)", nil)
	default:
		slog.Error("schedule_unhandled_error", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Terjadi kesalahan. Silakan coba lagi.", nil)
	}
}

func toData(s *Schedule) gin.H {
	return gin.H{
		"id":            s.ID,
		"employee_id":   s.EmployeeID,
		"employee_name": s.EmployeeName,
		"shift_id":      s.ShiftID,
		"shift_name":    s.ShiftName,
		"day_of_week":   s.DayOfWeek,
		"created_at":    s.CreatedAt,
		"updated_at":    s.UpdatedAt,
	}
}
