package user

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

// CreateRequest's Password is optional: leave it empty to have the server
// generate one, returned once in the response for the creating admin to
// hand to the new user out of band.
type CreateRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=150"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Role     string `json:"role" validate:"required"`
	Password string `json:"password" validate:"omitempty,min=8,max=100"`
}

type UpdateRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=150"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Role     string `json:"role" validate:"required"`
	IsActive bool   `json:"is_active"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// actor resolves the audit-trail Actor for the currently authenticated
// request straight from the JWT claims middleware.AuthRequired already put
// in context — no database lookup needed, since the token itself carries
// the actor's name (see pkg/jwt.Claims' doc comment).
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

	u, generatedPassword, err := h.service.Create(
		c.Request.Context(), h.actor(c), req.Name, req.Email, rbac.Role(req.Role), req.Password,
	)
	if err != nil {
		writeError(c, err)
		return
	}

	data := toData(u)
	if generatedPassword != "" {
		data["generated_password"] = generatedPassword
	}
	response.OK(c, http.StatusCreated, "Akun berhasil dibuat", data)
}

func (h *Handler) List(c *gin.Context) {
	p := pagination.FromQuery(c.Request.URL.Query())
	items, total, err := h.service.List(c.Request.Context(), p)
	if err != nil {
		slog.Error("user_list_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil data akun", nil)
		return
	}

	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, toData(&items[i]))
	}
	response.OK(c, http.StatusOK, "Daftar akun", gin.H{
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

	u, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Detail akun", toData(u))
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

	u, err := h.service.Update(c.Request.Context(), h.actor(c), id, req.Name, req.Email, rbac.Role(req.Role), req.IsActive)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Akun berhasil diperbarui", toData(u))
}

func (h *Handler) ResetPassword(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	password, err := h.service.ResetPassword(c.Request.Context(), h.actor(c), id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, "Password berhasil direset", gin.H{"generated_password": password})
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
	response.OK(c, http.StatusOK, "Akun berhasil dinonaktifkan", nil)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "Akun tidak ditemukan", nil)
	case errors.Is(err, ErrEmailTaken):
		response.Fail(c, http.StatusConflict, "Email sudah digunakan", nil)
	case errors.Is(err, ErrInvalidRole):
		response.Fail(c, http.StatusUnprocessableEntity, "Peran tidak valid", nil)
	default:
		slog.Error("user_unhandled_error", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Terjadi kesalahan. Silakan coba lagi.", nil)
	}
}

func toData(u *User) gin.H {
	return gin.H{
		"id":         u.ID,
		"name":       u.Name,
		"email":      u.Email,
		"role":       u.Role,
		"is_active":  u.IsActive,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
}
