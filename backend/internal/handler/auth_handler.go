package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ahmadqo/digital-achievement-ledger/internal/middleware"
	"github.com/ahmadqo/digital-achievement-ledger/internal/response"
	"github.com/ahmadqo/digital-achievement-ledger/internal/service"
	"github.com/ahmadqo/digital-achievement-ledger/internal/utils"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login godoc
// POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req service.LoginRequest

	if err := utils.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Format request tidak valid", err.Error())
		return
	}

	// Validasi input
	errs := utils.ValidationErrors{}
	req.Email = utils.SanitizeString(strings.ToLower(req.Email))

	if req.Email == "" {
		errs["email"] = "Email wajib diisi"
	} else if !utils.IsValidEmail(req.Email) {
		errs["email"] = "Format email tidak valid"
	}
	if req.Password == "" {
		errs["password"] = "Password wajib diisi"
	}

	if errs.HasErrors() {
		response.BadRequest(w, "Validasi gagal", errs)
		return
	}

	// Proses login
	result, err := h.authService.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			response.Unauthorized(w, err.Error())
		case errors.Is(err, service.ErrAccountDisabled):
			response.Forbidden(w, err.Error())
		default:
			response.InternalError(w, "Terjadi kesalahan server")
		}
		return
	}

	response.Success(w, "Login berhasil", result)
}

// Register godoc
// POST /api/v1/auth/register
// Hanya admin yang bisa register user baru (dikontrol di route)
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterRequest

	if err := utils.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Format request tidak valid", err.Error())
		return
	}

	// Validasi
	errs := utils.ValidationErrors{}
	req.Name = utils.SanitizeString(req.Name)
	req.Email = utils.SanitizeString(strings.ToLower(req.Email))

	if req.Name == "" {
		errs["name"] = "Nama wajib diisi"
	}
	if req.Email == "" {
		errs["email"] = "Email wajib diisi"
	} else if !utils.IsValidEmail(req.Email) {
		errs["email"] = "Format email tidak valid"
	}
	if req.Password == "" {
		errs["password"] = "Password wajib diisi"
	} else if !utils.IsValidPassword(req.Password) {
		errs["password"] = "Password minimal 8 karakter dan harus mengandung huruf dan angka"
	}

	validRoles := map[string]bool{"operator": true, "admin": true, "headmaster": true}
	if req.Role != "" && !validRoles[string(req.Role)] {
		errs["role"] = "Role tidak valid (operator, admin, headmaster)"
	}

	if errs.HasErrors() {
		response.BadRequest(w, "Validasi gagal", errs)
		return
	}

	result, err := h.authService.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			response.BadRequest(w, err.Error(), nil)
			return
		}
		response.InternalError(w, "Terjadi kesalahan server")
		return
	}

	response.Created(w, "User berhasil dibuat", result)
}

// RefreshToken godoc
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req service.RefreshTokenRequest

	if err := utils.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, "Format request tidak valid", err.Error())
		return
	}

	if req.RefreshToken == "" {
		response.BadRequest(w, "Refresh token wajib diisi", nil)
		return
	}

	tokenPair, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}

	response.Success(w, "Token berhasil diperbarui", tokenPair)
}

// Me godoc
// GET /api/v1/auth/me (protected)
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		response.Unauthorized(w, "User tidak terautentikasi")
		return
	}

	user, err := h.authService.Me(r.Context(), userID)
	if err != nil {
		response.NotFound(w, "User tidak ditemukan")
		return
	}

	response.Success(w, "Data user berhasil diambil", user)
}