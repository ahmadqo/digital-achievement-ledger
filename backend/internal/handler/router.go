package handler

import (
	"net/http"

	appMiddleware "github.com/ahmadqo/digital-achievement-ledger/internal/middleware"
	"github.com/ahmadqo/digital-achievement-ledger/internal/response"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Router struct {
	authHandler *AuthHandler
	jwtSecret   string
}

func NewRouter(authHandler *AuthHandler, jwtSecret string) *Router {
	return &Router{
		authHandler: authHandler,
		jwtSecret:   jwtSecret,
	}
}

func (ro *Router) Setup() http.Handler {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, "Server berjalan dengan baik", map[string]string{
			"status": "ok",
		})
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {

		// Auth routes (public)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", ro.authHandler.Login)
			r.Post("/refresh", ro.authHandler.RefreshToken)

			// Protected auth routes
			r.Group(func(r chi.Router) {
				r.Use(appMiddleware.Authenticate(ro.jwtSecret))
				r.Get("/me", ro.authHandler.Me)
			})
		})

		// User management (admin only)
		r.Route("/users", func(r chi.Router) {
			r.Use(appMiddleware.Authenticate(ro.jwtSecret))
			r.Use(appMiddleware.RequireRole("admin", "headmaster"))
			r.Post("/", ro.authHandler.Register)
		})

		// Protected routes - akan diisi nanti
		r.Group(func(r chi.Router) {
			r.Use(appMiddleware.Authenticate(ro.jwtSecret))

			// Students
			r.Route("/students", func(r chi.Router) {
				// akan diisi di tahap berikutnya
			})

			// Achievements
			r.Route("/achievements", func(r chi.Router) {
				// akan diisi di tahap berikutnya
			})

			// Certificates
			r.Route("/certificates", func(r chi.Router) {
				// akan diisi di tahap berikutnya
			})
		})

		// Public verification endpoint
		r.Get("/verify/{token}", func(w http.ResponseWriter, r *http.Request) {
			// akan diisi di tahap berikutnya
			response.Success(w, "Endpoint verifikasi", nil)
		})
	})

	return r
}