package faceprofile

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/device"
	"github.com/suryaintigas/absensi-backend/pkg/response"
	"github.com/suryaintigas/absensi-backend/pkg/validator"
)

type EnrollRequest struct {
	FeatureVector []float64 `json:"feature_vector" validate:"required,min=1"`
}

// Handler serves both the dashboard-facing enrollment endpoint (JWT) and
// the tablet-facing sync endpoint (device_code) — see their doc comments
// for why each uses a different trust boundary.
type Handler struct {
	service    *Service
	deviceRepo device.Repository
}

func NewHandler(service *Service, deviceRepo device.Repository) *Handler {
	return &Handler{service: service, deviceRepo: deviceRepo}
}

// Enroll handles PUT /api/v1/employees/:id/face-profile. Requires
// SUPER_ADMIN/ADMIN/HR (wired in cmd/server) — capturing someone's face is
// an HR action performed on the tablet's camera (the only camera in the
// system) after that staff member logs into the tablet.
func (h *Handler) Enroll(c *gin.Context) {
	employeeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID karyawan tidak valid", nil)
		return
	}

	var req EnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	fp, err := h.service.Enroll(c.Request.Context(), employeeID, req.FeatureVector)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Profil wajah berhasil disimpan", toData(fp))
}

// Sync handles GET /api/v1/face-profiles/sync?device_code=... — public,
// gated by a registered+active device_code like Phase 4's attendance
// endpoints, since this is what lets the tablet run face recognition
// entirely on-device (including offline) instead of sending images to the
// server per request.
func (h *Handler) Sync(c *gin.Context) {
	code := c.Query("device_code")
	if code == "" {
		response.Fail(c, http.StatusBadRequest, "device_code wajib diisi", nil)
		return
	}

	dev, err := h.deviceRepo.FindByCode(c.Request.Context(), code)
	if err != nil || dev.Status != device.StatusActive {
		response.Fail(c, http.StatusForbidden, "Perangkat tidak terdaftar atau tidak aktif", nil)
		return
	}

	profiles, err := h.service.ListActive(c.Request.Context())
	if err != nil {
		slog.Error("faceprofile_sync_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil data profil wajah", nil)
		return
	}

	list := make([]gin.H, 0, len(profiles))
	for i := range profiles {
		list = append(list, toData(&profiles[i]))
	}
	response.OK(c, http.StatusOK, "Sinkronisasi profil wajah", gin.H{"items": list})
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "Profil wajah tidak ditemukan", nil)
	case errors.Is(err, ErrInvalidEmployee):
		response.Fail(c, http.StatusUnprocessableEntity, "Karyawan tidak ditemukan", nil)
	case errors.Is(err, ErrEmptyVector):
		response.Fail(c, http.StatusUnprocessableEntity, "Data wajah tidak valid", nil)
	default:
		slog.Error("faceprofile_unhandled_error", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Terjadi kesalahan. Silakan coba lagi.", nil)
	}
}

func toData(fp *FaceProfile) gin.H {
	return gin.H{
		"employee_id":     fp.EmployeeID,
		"employee_name":   fp.EmployeeName,
		"employee_number": fp.EmployeeNumber,
		"feature_vector":  fp.FeatureVector,
		"updated_at":      fp.UpdatedAt,
	}
}
