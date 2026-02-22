package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ahmadqo/digital-achievement-ledger/internal/response"
	"github.com/ahmadqo/digital-achievement-ledger/internal/utils"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyEmail  contextKey = "email"
	ContextKeyRole   contextKey = "role"
	ContextKeyName   contextKey = "name"
)

// Authenticate memvalidasi JWT dari Authorization header
func Authenticate(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "Token tidak ditemukan")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.Unauthorized(w, "Format token tidak valid, gunakan: Bearer <token>")
				return
			}

			tokenString := parts[1]
			claims, err := utils.ValidateToken(tokenString, jwtSecret)
			if err != nil {
				response.Unauthorized(w, "Token tidak valid atau sudah expired")
				return
			}

			// Simpan claims ke context
			ctx := r.Context()
			ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)
			ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
			ctx = context.WithValue(ctx, ContextKeyName, claims.Name)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole memastikan user memiliki salah satu dari role yang diizinkan
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, ok := r.Context().Value(ContextKeyRole).(string)
			if !ok || userRole == "" {
				response.Unauthorized(w, "Role tidak ditemukan dalam token")
				return
			}

			for _, role := range roles {
				if strings.EqualFold(userRole, role) {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Forbidden(w, "Anda tidak memiliki akses ke resource ini")
		})
	}
}

// GetUserIDFromContext helper untuk ambil user ID dari context
func GetUserIDFromContext(ctx context.Context) string {
	val, _ := ctx.Value(ContextKeyUserID).(string)
	return val
}

func GetRoleFromContext(ctx context.Context) string {
	val, _ := ctx.Value(ContextKeyRole).(string)
	return val
}