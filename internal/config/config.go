package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort    string
	DatabaseURL   string
	MigrateURL    string
	MinPoolSize   int64
	MaxPoolSize   int64
	JWTSecret     string
	UploadDir     string
	PublicBaseURL string
	MaxUploadSize int64
}

func Load() *Config {
	JWTSecret := getEnv("JWT_SECRET", "")
	if JWTSecret == "" {
		panic("JWT_SECRET is required")
	}

	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "3001"),
		DatabaseURL:   getEnv("POSTGRES_URL", ""),
		MigrateURL:    getEnv("MIGRATE_URL", ""),
		MinPoolSize:   getEnvInt64("MIN_POOL_SIZE", 5),
		MaxPoolSize:   getEnvInt64("MAX_POOL_SIZE", 25),
		JWTSecret:     JWTSecret,
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		PublicBaseURL: getEnv("PUBLIC_BASE_URL", "http://localhost:3000"),
		MaxUploadSize: getEnvInt64("MAX_UPLOAD_SIZE", 5*1024*1024),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
}