package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ahmadqo/digital-achievement-ledger/internal/config"
	"github.com/ahmadqo/digital-achievement-ledger/internal/database"
	"github.com/ahmadqo/digital-achievement-ledger/internal/handler"
	"github.com/ahmadqo/digital-achievement-ledger/internal/repository"
	"github.com/ahmadqo/digital-achievement-ledger/internal/service"
	"github.com/ahmadqo/digital-achievement-ledger/internal/utils"
)

// @title           Digital Achievement Ledger API
// @version         1.0
// @description     This is a backend server for Digital Achievement Ledger.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()

	// ── Database ─────────────────────────────────────
	db := database.Connect(&cfg.Database)
	defer db.Close()

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "./migrations"
	}
	log.Printf("Running migrations from: %s", migrationsPath)
	if err := database.RunMigrations(db, migrationsPath); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	seeder := database.NewSeeder(db)
	if err := seeder.SeedAdminUser(context.Background()); err != nil {
		log.Printf("Warning: seed failed: %v", err)
	}

	// ── Storage (MinIO) ───────────────────────────────
	storage, err := utils.NewStorageService(&cfg.MinIO)
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}
	log.Println("MinIO connected successfully")

	// ── Repositories ─────────────────────────────────
	userRepo := repository.NewUserRepository(db)
	studentRepo := repository.NewStudentRepository(db)
	achievementRepo := repository.NewAchievementRepository(db)
	certificateRepo := repository.NewCertificateRepository(db)

	// ── Services ─────────────────────────────────────
	authService := service.NewAuthService(userRepo, cfg)
	studentService := service.NewStudentService(studentRepo, storage)
	achievementService := service.NewAchievementService(achievementRepo, studentRepo, storage)
	certificateService := service.NewCertificateService(certificateRepo, studentRepo, achievementRepo, storage)

	// ── Handlers ─────────────────────────────────────
	authHandler := handler.NewAuthHandler(authService)
	studentHandler := handler.NewStudentHandler(studentService)
	achievementHandler := handler.NewAchievementHandler(achievementService)
	certificateHandler := handler.NewCertificateHandler(certificateService)

	// ── Router ────────────────────────────────────────
	router := handler.NewRouter(
		authHandler,
		studentHandler,
		achievementHandler,
		certificateHandler,
		cfg.JWT.Secret,
	)

	// ── HTTP Server ───────────────────────────────────
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.App.Port),
		Handler:      router.Setup(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 Server berjalan di port %s (mode: %s)", cfg.App.Port, cfg.App.Env)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server stopped gracefully")
}
