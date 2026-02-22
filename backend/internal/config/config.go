package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	JWT      JWTConfig
	MinIO    MinIOConfig
}

type AppConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type JWTConfig struct {
	Secret          string
	ExpireHours     int
	RefreshExpHours int
}

type MinIOConfig struct {
	Endpoint  string
	User      string
	Password  string
	Bucket    string
	UseSSL    bool
}

func Load() *Config {
	// Load .env jika ada (development), di production pakai env variable langsung
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}

	jwtExpire, _ := strconv.Atoi(getEnv("JWT_EXPIRE_HOURS", "24"))
	jwtRefreshExpire, _ := strconv.Atoi(getEnv("JWT_REFRESH_EXPIRE_HOURS", "168"))
	minioSSL, _ := strconv.ParseBool(getEnv("MINIO_USE_SSL", "false"))

	return &Config{
		App: AppConfig{
			Port: getEnv("APP_PORT", "8080"),
			Env:  getEnv("APP_ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "skp_user"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "skp_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "change-this-secret"),
			ExpireHours:     jwtExpire,
			RefreshExpHours: jwtRefreshExpire,
		},
		MinIO: MinIOConfig{
			Endpoint: getEnv("MINIO_ENDPOINT", "localhost:9000"),
			User:     getEnv("MINIO_USER", "minioadmin"),
			Password: getEnv("MINIO_PASSWORD", "minioadmin123"),
			Bucket:   getEnv("MINIO_BUCKET", "skp-attachments"),
			UseSSL:   minioSSL,
		},
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}