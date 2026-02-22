package database

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type Seeder struct {
	db *sqlx.DB
}

func NewSeeder(db *sqlx.DB) *Seeder {
	return &Seeder{db: db}
}

// SeedAdminUser membuat user admin default jika belum ada
func (s *Seeder) SeedAdminUser(ctx context.Context) error {
	// Cek apakah sudah ada admin
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Println("Admin user already exists, skipping seed")
		return nil
	}

	// Hash password default
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Admin@123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (id, name, email, password, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`,
		uuid.New(),
		"Administrator",
		"admin@sekolah.sch.id",
		string(hashedPassword),
		"admin",
		true,
	)

	if err != nil {
		return err
	}

	log.Println("✅ Default admin user created:")
	log.Println("   Email   : admin@sekolah.sch.id")
	log.Println("   Password: Admin@123")
	log.Println("   ⚠️  Segera ganti password setelah login pertama!")

	return nil
}