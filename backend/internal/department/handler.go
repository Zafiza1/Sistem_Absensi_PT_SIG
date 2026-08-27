package department

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
	Name        string `json:"name" validate:"required,min=2,max=150"`
	Description string `json:"description" validate:"max=1000"`
}

type UpdateRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=150"`
	Description string `json:"description" validate:"max=1000"`
	IsActive    bool   `json:"is_active"`
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

	d, err := h.service.Create(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusCreated, "Departemen berhasil dibuat", toData(d))
}

func (h *Handler) List(c *gin.Context) {
	p := pagination.FromQuery(c.Request.URL.Query())
	items, total, err := h.service.List(c.Request.Context(), p)
	if err != nil {
		slog.Error("department_list_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil data departemen", nil)
		return
	}

	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, toData(&items[i]))
	}
	response.OK(c, http.StatusOK, "Daftar departemen", gin.H{
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
	response.OK(c, http.StatusOK, "Detail departemen", toData(d))
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

	d, err := h.service.Update(c.Request.Context(), id, req.Name, req.Description, req.IsActive)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Departemen berhasil diperbarui", toData(d))
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
	response.OK(c, http.StatusOK, "Departemen berhasil dihapus", nil)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "Departemen tidak ditemukan", nil)
	case errors.Is(err, ErrNameTaken):
		response.Fail(c, http.StatusConflict, "Nama departemen sudah digunakan", nil)
	case errors.Is(err, ErrHasReferences):
		response.Fail(c, http.StatusConflict, "Departemen masih digunakan oleh karyawan aktif", nil)
	default:
		slog.Error("department_unhandled_error", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Terjadi kesalahan. Silakan coba lagi.", nil)
	}
}

func toData(d *Department) gin.H {
	return gin.H{
		"id":          d.ID,
		"name":        d.Name,
		"description": d.Description,
		"is_active":   d.IsActive,
		"created_at":  d.CreatedAt,
		"updated_at":  d.UpdatedAt,
	}
}
