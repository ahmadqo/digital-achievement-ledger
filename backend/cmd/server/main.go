package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/ahmadqo/digital-achievement-ledger/internal/config"
	"github.com/ahmadqo/digital-achievement-ledger/internal/database"
	"github.com/ahmadqo/digital-achievement-ledger/internal/handler"
	"github.com/ahmadqo/digital-achievement-ledger/internal/repository"
	"github.com/ahmadqo/digital-achievement-ledger/internal/service"
)

func main() {
	// Load config
	cfg := config.Load()

	// Connect database
	db := database.Connect(&cfg.Database)
	defer db.Close()

	// Jalankan migrations
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	migrationsPath := filepath.Join(projectRoot, "migrations")

	// Fallback jika path tidak ditemukan (misal di dalam Docker)
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		migrationsPath = "./migrations"
	}

	if err := database.RunMigrations(db, migrationsPath); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Seed data default
	seeder := database.NewSeeder(db)
	if err := seeder.SeedAdminUser(context.Background()); err != nil {
		log.Printf("Warning: seed failed: %v", err)
	}

	// Wire dependencies
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, cfg)
	authHandler := handler.NewAuthHandler(authService)

	// Setup router
	router := handler.NewRouter(authHandler, cfg.JWT.Secret)
	httpHandler := router.Setup()

	// HTTP Server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.App.Port),
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
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