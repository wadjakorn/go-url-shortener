package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	AppEnv      string
	BaseURL     string
}

func Load() *Config {
	_ = godotenv.Load() // Ignore error if .env not found (e.g. prod)

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "file:db.sqlite"),
		AppEnv:      getEnv("APP_ENV", "local"),
		BaseURL:     getEnv("BASE_URL", "http://localhost:8080"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
