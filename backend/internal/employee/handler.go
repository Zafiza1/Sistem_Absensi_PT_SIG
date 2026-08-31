package employee

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/middleware"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
	"github.com/suryaintigas/absensi-backend/pkg/rbac"
	"github.com/suryaintigas/absensi-backend/pkg/response"
	"github.com/suryaintigas/absensi-backend/pkg/validator"
)

type CreateRequest struct {
	EmployeeNumber string     `json:"employee_number" validate:"required,min=1,max=50"`
	Name           string     `json:"name" validate:"required,min=2,max=150"`
	Email          *string    `json:"email" validate:"omitempty,email"`
	Phone          *string    `json:"phone" validate:"omitempty,max=30"`
	DepartmentID   *uuid.UUID `json:"department_id"`
	PositionID     *uuid.UUID `json:"position_id"`
	ShiftID        *uuid.UUID `json:"shift_id"`
}

type UpdateRequest struct {
	CreateRequest
	Status string `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
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

	e, err := h.service.Create(c.Request.Context(), h.actor(c), Input{
		EmployeeNumber: req.EmployeeNumber,
		Name:           req.Name,
		Email:          req.Email,
		Phone:          req.Phone,
		DepartmentID:   req.DepartmentID,
		PositionID:     req.PositionID,
		ShiftID:        req.ShiftID,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusCreated, "Karyawan berhasil ditambahkan", toData(e))
}

func (h *Handler) List(c *gin.Context) {
	q := c.Request.URL.Query()
	p := pagination.FromQuery(q)

	f := Filter{
		Search: q.Get("search"),
		Status: q.Get("status"),
	}
	if raw := q.Get("department_id"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			f.DepartmentID = &id
		}
	}

	items, total, err := h.service.List(c.Request.Context(), f, p)
	if err != nil {
		slog.Error("employee_list_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil data karyawan", nil)
		return
	}

	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, toData(&items[i]))
	}
	response.OK(c, http.StatusOK, "Daftar karyawan", gin.H{
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

	e, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Detail karyawan", toData(e))
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

	e, err := h.service.Update(c.Request.Context(), h.actor(c), id, Input{
		EmployeeNumber: req.EmployeeNumber,
		Name:           req.Name,
		Email:          req.Email,
		Phone:          req.Phone,
		DepartmentID:   req.DepartmentID,
		PositionID:     req.PositionID,
		ShiftID:        req.ShiftID,
		Status:         req.Status,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Karyawan berhasil diperbarui", toData(e))
}

// Delete deactivates (soft-deletes) an employee — see Service.Deactivate.
func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	if err := h.service.Deactivate(c.Request.Context(), h.actor(c), id); err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Karyawan berhasil dinonaktifkan", nil)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "Karyawan tidak ditemukan", nil)
	case errors.Is(err, ErrEmployeeNumberTaken):
		response.Fail(c, http.StatusConflict, "Nomor karyawan sudah digunakan", nil)
	case errors.Is(err, ErrEmailTaken):
		response.Fail(c, http.StatusConflict, "Email sudah digunakan", nil)
	case errors.Is(err, ErrInvalidReference):
		response.Fail(c, http.StatusUnprocessableEntity, "Departemen, jabatan, atau shift tidak ditemukan", nil)
	default:
		slog.Error("employee_unhandled_error", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Terjadi kesalahan. Silakan coba lagi.", nil)
	}
}

func toData(e *Employee) gin.H {
	return gin.H{
		"id":              e.ID,
		"employee_number": e.EmployeeNumber,
		"name":            e.Name,
		"email":           e.Email,
		"phone":           e.Phone,
		"department_id":   e.DepartmentID,
		"department_name": e.DepartmentName,
		"position_id":     e.PositionID,
		"position_name":   e.PositionName,
		"shift_id":        e.ShiftID,
		"shift_name":      e.ShiftName,
		"status":          e.Status,
		"created_at":      e.CreatedAt,
		"updated_at":      e.UpdatedAt,
	}
}
