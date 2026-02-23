package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/ahmadqo/digital-achievement-ledger/docs" // Import generated docs
	appMiddleware "github.com/ahmadqo/digital-achievement-ledger/internal/middleware"
	"github.com/ahmadqo/digital-achievement-ledger/internal/response"
)

type Router struct {
	authHandler        *AuthHandler
	studentHandler     *StudentHandler
	achievementHandler *AchievementHandler
	certificateHandler *CertificateHandler
	jwtSecret          string
}

func NewRouter(
	authHandler *AuthHandler,
	studentHandler *StudentHandler,
	achievementHandler *AchievementHandler,
	certificateHandler *CertificateHandler,
	jwtSecret string,
) *Router {
	return &Router{
		authHandler:        authHandler,
		studentHandler:     studentHandler,
		achievementHandler: achievementHandler,
		certificateHandler: certificateHandler,
		jwtSecret:          jwtSecret,
	}
}

func (ro *Router) Setup() http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, "Server berjalan dengan baik", map[string]string{"status": "ok"})
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {

		// ── Auth (public) ────────────────────────────────
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", ro.authHandler.Login)
			r.Post("/refresh", ro.authHandler.RefreshToken)

			r.Group(func(r chi.Router) {
				r.Use(appMiddleware.Authenticate(ro.jwtSecret))
				r.Get("/me", ro.authHandler.Me)
			})
		})

		// ── Public: verifikasi QR ─────────────────────────
		r.Get("/verify/{token}", ro.certificateHandler.Verify)

		// ── Protected routes ──────────────────────────────
		r.Group(func(r chi.Router) {
			r.Use(appMiddleware.Authenticate(ro.jwtSecret))

			// User management (admin only)
			r.Route("/users", func(r chi.Router) {
				r.Use(appMiddleware.RequireRole("admin", "headmaster"))
				r.Post("/", ro.authHandler.Register)
			})

			// Students
			r.Route("/students", func(r chi.Router) {
				r.Get("/", ro.studentHandler.GetAll)
				r.Post("/", ro.studentHandler.Create)
				r.Get("/{id}", ro.studentHandler.GetByID)
				r.Put("/{id}", ro.studentHandler.Update)
				r.Delete("/{id}", ro.studentHandler.Delete)
				r.Post("/{id}/photo", ro.studentHandler.UploadPhoto)
			})

			// Achievements
			r.Route("/achievements", func(r chi.Router) {
				r.Get("/categories", ro.achievementHandler.GetCategories)
				r.Get("/levels", ro.achievementHandler.GetLevels)
				r.Get("/", ro.achievementHandler.GetAll)
				r.Post("/", ro.achievementHandler.Create)
				r.Get("/{id}", ro.achievementHandler.GetByID)
				r.Put("/{id}", ro.achievementHandler.Update)
				r.Delete("/{id}", ro.achievementHandler.Delete)
				r.Post("/{id}/attachments", ro.achievementHandler.UploadAttachment)
				r.Delete("/attachments/{attachmentId}", ro.achievementHandler.DeleteAttachment)
			})

			// Certificates
			r.Route("/certificates", func(r chi.Router) {
				r.Get("/", ro.certificateHandler.GetAll)
				r.Post("/", ro.certificateHandler.Create)
				r.Get("/{id}", ro.certificateHandler.GetByID)
				r.Get("/{id}/download", ro.certificateHandler.Download)
				r.Post("/{id}/revoke", ro.certificateHandler.Revoke)
			})
		})
	})

	return r
}
