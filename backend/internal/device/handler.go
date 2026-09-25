package device

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

type RegisterRequest struct {
	DeviceName   string                 `json:"device_name" validate:"required,min=2,max=150"`
	DeviceCode   string                 `json:"device_code" validate:"required,min=2,max=100"`
	Location     string                 `json:"location" validate:"max=255"`
	AppVersion   string                 `json:"app_version" validate:"max=50"`
	DeviceType   string                 `json:"device_type" validate:"omitempty,oneof=FINGERSPOT TABLET OTHER"`
	SerialNumber string                 `json:"serial_number" validate:"max=100"`
	IPAddress    string                 `json:"ip_address" validate:"omitempty,ip"`
	Port         int                    `json:"port" validate:"omitempty,min=0,max=65535"`
	DeviceConfig map[string]interface{} `json:"device_config"`
}

type UpdateRequest struct {
	RegisterRequest
	Status           string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	ConnectionStatus string `json:"connection_status" validate:"omitempty,oneof=CONNECTED DISCONNECTED ERROR SYNCING"`
	SyncStatus       string `json:"sync_status" validate:"omitempty,oneof=IDLE SYNCING SUCCESS FAILED"`
	ErrorMessage     string `json:"error_message" validate:"max=500"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// actor resolves the audit-trail Actor for the currently authenticated
// request straight from the JWT claims middleware.AuthRequired already put
// in context — see user.Handler.actor's doc comment for why this needs no
// database lookup.
func (h *Handler) actor(c *gin.Context) Actor {
	id, _ := uuid.Parse(c.GetString(middleware.ContextKeyUserID))
	role := rbac.Role(c.GetString(middleware.ContextKeyUserRole))
	name := c.GetString(middleware.ContextKeyUserName)
	return Actor{ID: id, Name: name, Role: role, IP: c.ClientIP()}
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

	d, err := h.service.Register(c.Request.Context(), h.actor(c), Input{
		DeviceName:   req.DeviceName,
		DeviceCode:   req.DeviceCode,
		Location:     req.Location,
		AppVersion:   req.AppVersion,
		DeviceType:   req.DeviceType,
		SerialNumber: req.SerialNumber,
		IPAddress:    req.IPAddress,
		Port:         req.Port,
		DeviceConfig: req.DeviceConfig,
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

// VerifyByCode is public (no JWT): the tablet calls it on launch, and
// periodically thereafter, to confirm it is still a registered, active
// device before showing the attendance screen — per the spec, an
// unregistered or deactivated tablet must never be allowed to proceed. It
// deliberately returns only non-sensitive display fields.
func (h *Handler) VerifyByCode(c *gin.Context) {
	code := c.Param("code")

	d, err := h.service.GetByCode(c.Request.Context(), code)
	if err != nil {
		writeError(c, err)
		return
	}
	if d.Status != StatusActive {
		response.Fail(c, http.StatusForbidden, "Perangkat tidak aktif. Hubungi administrator.", nil)
		return
	}

	response.OK(c, http.StatusOK, "Perangkat terverifikasi", gin.H{
		"device_name": d.DeviceName,
		"location":    d.Location,
		"status":      d.Status,
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

	d, err := h.service.Update(c.Request.Context(), h.actor(c), id, Input{
		DeviceName:   req.DeviceName,
		DeviceCode:   req.DeviceCode,
		Location:     req.Location,
		AppVersion:   req.AppVersion,
		DeviceType:   req.DeviceType,
		SerialNumber: req.SerialNumber,
		IPAddress:    req.IPAddress,
		Port:         req.Port,
		DeviceConfig: req.DeviceConfig,
	}, req.Status)
	if err != nil {
		writeError(c, err)
		return
	}

	// Update connection and sync status if provided
	if req.ConnectionStatus != "" || req.SyncStatus != "" || req.ErrorMessage != "" {
		if err := h.service.UpdateStatus(c.Request.Context(), h.actor(c), id, req.ConnectionStatus, req.SyncStatus, req.ErrorMessage); err != nil {
			writeError(c, err)
			return
		}
		// Refresh the device data to return updated status
		d, err = h.service.Get(c.Request.Context(), id)
		if err != nil {
			writeError(c, err)
			return
		}
	}

	response.OK(c, http.StatusOK, "Perangkat berhasil diperbarui", toData(d))
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	if err := h.service.Delete(c.Request.Context(), h.actor(c), id); err != nil {
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
		"id":                d.ID,
		"device_name":       d.DeviceName,
		"device_code":       d.DeviceCode,
		"location":          d.Location,
		"status":            d.Status,
		"device_type":       d.DeviceType,
		"serial_number":     d.SerialNumber,
		"ip_address":        d.IPAddress,
		"port":              d.Port,
		"connection_status": d.ConnectionStatus,
		"app_version":       d.AppVersion,
		"last_seen_at":      d.LastSeenAt,
		"last_sync_at":      d.LastSyncAt,
		"sync_status":       d.SyncStatus,
		"error_message":     d.ErrorMessage,
		"device_config":     d.DeviceConfig,
		"is_online":         d.IsOnline(time.Now()),
		"is_connected":      d.IsConnected(),
		"is_syncing":        d.IsSyncing(),
		"created_at":        d.CreatedAt,
		"updated_at":        d.UpdatedAt,
	}
}
