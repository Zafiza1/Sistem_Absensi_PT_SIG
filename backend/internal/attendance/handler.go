package attendance

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/pagination"
	"github.com/suryaintigas/absensi-backend/pkg/response"
	"github.com/suryaintigas/absensi-backend/pkg/validator"
)

// CheckInRequest / CheckOutRequest are submitted by the tablet. EmployeeID
// is what Phase 5's on-device face recognition resolves to; DeviceCode is
// the tablet's own registered identifier (see internal/device).
type CheckInRequest struct {
	EmployeeID uuid.UUID `json:"employee_id" validate:"required"`
	DeviceCode string    `json:"device_code" validate:"required"`
}

type CheckOutRequest struct {
	EmployeeID uuid.UUID `json:"employee_id" validate:"required"`
	DeviceCode string    `json:"device_code" validate:"required"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CheckIn(c *gin.Context) {
	var req CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	att, err := h.service.CheckIn(c.Request.Context(), req.EmployeeID, req.DeviceCode)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusCreated, "Absensi berhasil", checkInData(att))
}

func (h *Handler) CheckOut(c *gin.Context) {
	var req CheckOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	att, err := h.service.CheckOut(c.Request.Context(), req.EmployeeID, req.DeviceCode)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Check-out berhasil", checkOutData(att))
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	att, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Detail absensi", fullData(att))
}

func (h *Handler) List(c *gin.Context) {
	q := c.Request.URL.Query()
	p := pagination.FromQuery(q)

	f := Filter{Status: q.Get("status")}
	if raw := q.Get("employee_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "employee_id tidak valid", nil)
			return
		}
		f.EmployeeID = &id
	}
	if raw := q.Get("date_from"); raw != "" {
		d, err := time.Parse("2006-01-02", raw)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "date_from harus format YYYY-MM-DD", nil)
			return
		}
		f.DateFrom = &d
	}
	if raw := q.Get("date_to"); raw != "" {
		d, err := time.Parse("2006-01-02", raw)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "date_to harus format YYYY-MM-DD", nil)
			return
		}
		f.DateTo = &d
	}

	items, total, err := h.service.List(c.Request.Context(), f, p)
	if err != nil {
		slog.Error("attendance_list_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil data absensi", nil)
		return
	}

	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, fullData(&items[i]))
	}
	response.OK(c, http.StatusOK, "Riwayat absensi", gin.H{
		"items": list,
		"meta":  pagination.NewMeta(p, total),
	})
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrEmployeeNotFound):
		response.Fail(c, http.StatusNotFound, "Karyawan tidak ditemukan atau tidak aktif", nil)
	case errors.Is(err, ErrDeviceNotRegistered):
		response.Fail(c, http.StatusForbidden, "Perangkat tidak terdaftar atau tidak aktif", nil)
	case errors.Is(err, ErrNoShiftAssigned):
		response.Fail(c, http.StatusUnprocessableEntity, "Karyawan belum memiliki shift atau jadwal", nil)
	case errors.Is(err, ErrAlreadyCheckedIn):
		response.Fail(c, http.StatusConflict, "Anda sudah melakukan absensi hari ini", nil)
	case errors.Is(err, ErrNoOpenCheckIn):
		response.Fail(c, http.StatusConflict, "Belum ada check-in aktif untuk karyawan ini", nil)
	case errors.Is(err, ErrAlreadyCheckedOut):
		response.Fail(c, http.StatusConflict, "Absensi ini sudah check-out", nil)
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "Data absensi tidak ditemukan", nil)
	default:
		slog.Error("attendance_unhandled_error", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Terjadi kesalahan. Silakan coba lagi.", nil)
	}
}

// checkInData mirrors the spec's example check-in response shape
// (employee_name/type/time/status) plus a few extra fields useful to a
// real client.
func checkInData(a *Attendance) gin.H {
	return gin.H{
		"id":              a.ID,
		"employee_id":     a.EmployeeID,
		"employee_name":   a.EmployeeName,
		"type":            "check_in",
		"attendance_date": a.AttendanceDate.Format("2006-01-02"),
		"time":            a.CheckInAt.In(jakarta).Format("15:04:05"),
		"status":          strings.ToLower(a.Status),
		"late_minutes":    a.LateMinutes,
	}
}

func checkOutData(a *Attendance) gin.H {
	data := gin.H{
		"id":              a.ID,
		"employee_id":     a.EmployeeID,
		"employee_name":   a.EmployeeName,
		"type":            "check_out",
		"attendance_date": a.AttendanceDate.Format("2006-01-02"),
		"time":            a.CheckOutAt.In(jakarta).Format("15:04:05"),
		"status":          strings.ToLower(a.Status),
	}
	if a.WorkingDurationMinutes != nil {
		data["working_duration_minutes"] = *a.WorkingDurationMinutes
	}
	return data
}

// fullData is used for Get/List, where the dashboard wants the complete
// record rather than the compact tablet-facing shape.
func fullData(a *Attendance) gin.H {
	data := gin.H{
		"id":                    a.ID,
		"employee_id":           a.EmployeeID,
		"employee_name":         a.EmployeeName,
		"employee_number":       a.EmployeeNumber,
		"shift_id":              a.ShiftID,
		"shift_name":            a.ShiftName,
		"attendance_date":       a.AttendanceDate.Format("2006-01-02"),
		"check_in_at":           a.CheckInAt,
		"check_in_device_name":  a.CheckInDeviceName,
		"check_out_at":          a.CheckOutAt,
		"check_out_device_name": a.CheckOutDeviceName,
		"status":                a.Status,
		"late_minutes":          a.LateMinutes,
		"created_at":            a.CreatedAt,
		"updated_at":            a.UpdatedAt,
	}
	if a.WorkingDurationMinutes != nil {
		data["working_duration_minutes"] = *a.WorkingDurationMinutes
	}
	return data
}
