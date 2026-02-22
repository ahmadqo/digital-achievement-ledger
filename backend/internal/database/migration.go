package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

// RunMigrations menjalankan semua file SQL di folder migrations secara berurutan
func RunMigrations(db *sqlx.DB, migrationsPath string) error {
	// Buat tabel tracking migration jika belum ada
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     VARCHAR(255) PRIMARY KEY,
			executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Baca semua file .sql
	files, err := filepath.Glob(filepath.Join(migrationsPath, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	sort.Strings(files) // pastikan urutan: 001_, 002_, dst

	for _, file := range files {
		version := filepath.Base(file)

		// Cek apakah sudah dijalankan
		var exists bool
		err := db.QueryRowContext(context.Background(),
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", version, err)
		}

		if exists {
			log.Printf("Migration %s already applied, skipping", version)
			continue
		}

		// Baca dan jalankan file SQL
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		// Skip komentar dan jalankan
		sql := string(content)
		if strings.TrimSpace(sql) == "" {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", version, err)
		}

		if _, err := tx.Exec(sql); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", version, err)
		}

		log.Printf("✅ Migration applied: %s", version)
	}

	return nil
}