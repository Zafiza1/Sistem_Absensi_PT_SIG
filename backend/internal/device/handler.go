package device

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/pagination"
	"github.com/suryaintigas/absensi-backend/pkg/response"
	"github.com/suryaintigas/absensi-backend/pkg/validator"
)

type RegisterRequest struct {
	DeviceName string `json:"device_name" validate:"required,min=2,max=150"`
	DeviceCode string `json:"device_code" validate:"required,min=2,max=100"`
	Location   string `json:"location" validate:"max=255"`
	AppVersion string `json:"app_version" validate:"max=50"`
}

type UpdateRequest struct {
	RegisterRequest
	Status string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	d, err := h.service.Register(c.Request.Context(), Input{
		DeviceName: req.DeviceName,
		DeviceCode: req.DeviceCode,
		Location:   req.Location,
		AppVersion: req.AppVersion,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusCreated, "Perangkat berhasil didaftarkan", toData(d))
}

func (h *Handler) List(c *gin.Context) {
	p := pagination.FromQuery(c.Request.URL.Query())
	items, total, err := h.service.List(c.Request.Context(), p)
	if err != nil {
		slog.Error("device_list_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil data perangkat", nil)
		return
	}

	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, toData(&items[i]))
	}
	response.OK(c, http.StatusOK, "Daftar perangkat", gin.H{
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

	d, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Detail perangkat", toData(d))
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

	d, err := h.service.Update(c.Request.Context(), id, Input{
		DeviceName: req.DeviceName,
		DeviceCode: req.DeviceCode,
		Location:   req.Location,
		AppVersion: req.AppVersion,
	}, req.Status)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Perangkat berhasil diperbarui", toData(d))
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
	response.OK(c, http.StatusOK, "Perangkat berhasil dihapus", nil)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "Perangkat tidak ditemukan", nil)
	case errors.Is(err, ErrDeviceCodeUsed):
		response.Fail(c, http.StatusConflict, "Kode perangkat sudah terdaftar", nil)
	default:
		slog.Error("device_unhandled_error", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Terjadi kesalahan. Silakan coba lagi.", nil)
	}
}

func toData(d *Device) gin.H {
	return gin.H{
		"id":           d.ID,
		"device_name":  d.DeviceName,
		"device_code":  d.DeviceCode,
		"location":     d.Location,
		"status":       d.Status,
		"app_version":  d.AppVersion,
		"last_seen_at": d.LastSeenAt,
		"last_sync_at": d.LastSyncAt,
		"is_online":    d.IsOnline(time.Now()),
		"created_at":   d.CreatedAt,
		"updated_at":   d.UpdatedAt,
	}
}
