package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/internal/middleware"
	"github.com/suryaintigas/absensi-backend/pkg/response"
	"github.com/suryaintigas/absensi-backend/pkg/validator"
)

// LoginRequest is the body of POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// RefreshRequest is the body of POST /api/v1/auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest is the body of POST /api/v1/auth/logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Handler wires HTTP requests to the auth Service.
type Handler struct {
	service *Service
}

// NewHandler builds a Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	pair, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		writeAuthError(c, err)
		return
	}

	response.OK(c, http.StatusOK, "Login berhasil", authData(pair, user))
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validasi gagal", validator.FormatErrors(err))
		return
	}

	pair, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		writeAuthError(c, err)
		return
	}

	response.OK(c, http.StatusOK, "Token berhasil diperbarui", authData(pair, user))
}

// Logout handles POST /api/v1/auth/logout.
func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Data permintaan tidak valid", nil)
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.Fail(c, http.StatusInternalServerError, "Gagal logout", nil)
		return
	}

	response.OK(c, http.StatusOK, "Logout berhasil", nil)
}

// Me handles GET /api/v1/auth/me. It requires middleware.AuthRequired to
// have already run and populated the authenticated user's ID in context.
func (h *Handler) Me(c *gin.Context) {
	id, err := uuid.Parse(c.GetString(middleware.ContextKeyUserID))
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, "Sesi tidak valid", nil)
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "Pengguna tidak ditemukan", nil)
		return
	}

	response.OK(c, http.StatusOK, "Profil pengguna", userData(user))
}

func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		response.Fail(c, http.StatusUnauthorized, "Email atau password salah", nil)
	case errors.Is(err, ErrAccountInactive):
		response.Fail(c, http.StatusForbidden, "Akun tidak aktif. Hubungi administrator.", nil)
	case errors.Is(err, ErrInvalidRefreshToken):
		response.Fail(c, http.StatusUnauthorized, "Refresh token tidak valid atau kedaluwarsa", nil)
	default:
		response.Fail(c, http.StatusInternalServerError, "Terjadi kesalahan. Silakan coba lagi.", nil)
	}
}

func authData(pair *TokenPair, user *User) gin.H {
	return gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    pair.ExpiresIn,
		"user":          userData(user),
	}
}

func userData(user *User) gin.H {
	return gin.H{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
	}
}
