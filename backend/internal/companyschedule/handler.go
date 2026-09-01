package companyschedule

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/middleware"
	"github.com/suryaintigas/absensi-backend/pkg/rbac"
	"github.com/suryaintigas/absensi-backend/pkg/response"
	"github.com/suryaintigas/absensi-backend/pkg/validator"
)

// SetRequest is the whole-week payload for PUT /company-schedule. The
// dashboard always sends all seven days; a day with "shift_id": null is a
// non-working day, and a day left out entirely is "not configured".
type SetRequest struct {
	Days []struct {
		DayOfWeek int        `json:"day_of_week" validate:"required,gte=1,lte=7"`
		ShiftID   *uuid.UUID `json:"shift_id"`
	} `json:"days" validate:"required,min=1,max=7,dive"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) actor(c *gin.Context) Actor {
	id, _ := uuid.Parse(c.GetString(middleware.ContextKeyUserID))
	return Actor{
		ID:   id,
		Name: c.GetString(middleware.ContextKeyUserName),
		Role: rbac.Role(c.GetString(middleware.ContextKeyUserRole)),
		IP:   c.ClientIP(),
	}
}

func (h *Handler) Get(c *gin.Context) {
	days, err := h.service.List(c.Request.Context())
	if err != nil {
		slog.Error("company_schedule_list_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil jadwal kerja", nil)
		return
	}
	response.OK(c, http.StatusOK, "Jadwal kerja perusahaan", gin.H{"days": toDataList(days)})
}

func (h *Handler) Set(c *gin.Context) {
	var req SetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	inputs := make([]DayInput, len(req.Days))
	for i, d := range req.Days {
		inputs[i] = DayInput{DayOfWeek: d.DayOfWeek, ShiftID: d.ShiftID}
	}

	days, err := h.service.Set(c.Request.Context(), h.actor(c), inputs)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Jadwal kerja berhasil disimpan", gin.H{"days": toDataList(days)})
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidDay):
		response.Fail(c, http.StatusUnprocessableEntity, "Hari harus antara 1 (Senin) - 7 (Minggu)", nil)
	case errors.Is(err, ErrDuplicateDay):
		response.Fail(c, http.StatusUnprocessableEntity, "Setiap hari hanya boleh muncul satu kali", nil)
	case errors.Is(err, ErrInvalidShift):
		response.Fail(c, http.StatusUnprocessableEntity, "Shift yang dipilih tidak ditemukan", nil)
	default:
		slog.Error("company_schedule_unhandled_error", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Terjadi kesalahan. Silakan coba lagi.", nil)
	}
}

func toDataList(days []Day) []gin.H {
	out := make([]gin.H, 0, len(days))
	for _, d := range days {
		out = append(out, gin.H{
			"day_of_week": d.DayOfWeek,
			"shift_id":    d.ShiftID,
			"shift_name":  d.ShiftName,
			"is_day_off":  d.ShiftID == nil,
		})
	}
	return out
}
