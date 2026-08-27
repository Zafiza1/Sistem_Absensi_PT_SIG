package shift

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
	Name                   string `json:"name" validate:"required,min=2,max=100"`
	StartTime              string `json:"start_time" validate:"required"`
	EndTime                string `json:"end_time" validate:"required"`
	LateToleranceMinutes   int    `json:"late_tolerance_minutes" validate:"gte=0"`
	WorkingDurationMinutes int    `json:"working_duration_minutes" validate:"gt=0"`
}

type UpdateRequest struct {
	CreateRequest
	IsActive bool `json:"is_active"`
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

	sh, err := h.service.Create(c.Request.Context(), Input{
		Name:                   req.Name,
		StartTime:              req.StartTime,
		EndTime:                req.EndTime,
		LateToleranceMinutes:   req.LateToleranceMinutes,
		WorkingDurationMinutes: req.WorkingDurationMinutes,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusCreated, "Shift berhasil dibuat", toData(sh))
}

func (h *Handler) List(c *gin.Context) {
	p := pagination.FromQuery(c.Request.URL.Query())
	items, total, err := h.service.List(c.Request.Context(), p)
	if err != nil {
		slog.Error("shift_list_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil data shift", nil)
		return
	}

	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, toData(&items[i]))
	}
	response.OK(c, http.StatusOK, "Daftar shift", gin.H{
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

	sh, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Detail shift", toData(sh))
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

	sh, err := h.service.Update(c.Request.Context(), id, Input{
		Name:                   req.Name,
		StartTime:              req.StartTime,
		EndTime:                req.EndTime,
		LateToleranceMinutes:   req.LateToleranceMinutes,
		WorkingDurationMinutes: req.WorkingDurationMinutes,
	}, req.IsActive)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Shift berhasil diperbarui", toData(sh))
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
	response.OK(c, http.StatusOK, "Shift berhasil dihapus", nil)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "Shift tidak ditemukan", nil)
	case errors.Is(err, ErrNameTaken):
		response.Fail(c, http.StatusConflict, "Nama shift sudah digunakan", nil)
	case errors.Is(err, ErrHasReferences):
		response.Fail(c, http.StatusConflict, "Shift masih digunakan oleh karyawan atau jadwal aktif", nil)
	case errors.Is(err, ErrInvalidTimeFormat):
		response.Fail(c, http.StatusUnprocessableEntity, "Format waktu harus HH:MM", nil)
	default:
		slog.Error("shift_unhandled_error", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Terjadi kesalahan. Silakan coba lagi.", nil)
	}
}

func toData(s *Shift) gin.H {
	return gin.H{
		"id":                       s.ID,
		"name":                     s.Name,
		"start_time":               s.StartTime,
		"end_time":                 s.EndTime,
		"is_overnight":             s.IsOvernight,
		"late_tolerance_minutes":   s.LateToleranceMinutes,
		"working_duration_minutes": s.WorkingDurationMinutes,
		"is_active":                s.IsActive,
		"created_at":               s.CreatedAt,
		"updated_at":               s.UpdatedAt,
	}
}
