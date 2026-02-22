package database

import (
	"fmt"
	"log"
	"time"

	"github.com/ahmadqo/digital-achievement-ledger/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver untuk database/sql
	"github.com/jmoiron/sqlx"
)

func Connect(cfg *config.DatabaseConfig) *sqlx.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	log.Println("Database connected successfully")
	return db
}