package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort   string
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
	DBSSLMode string
}

func Load() *Config {
	// Загружаем .env
	_ = godotenv.Load("config.env")

	return &Config{
		AppPort:   getEnv("APP_PORT", "8080"),
		DBHost:    getEnv("DB_HOST", "localhost"),
		DBPort:    getEnv("DB_PORT", "5433"),
		DBUser:    getEnv("DB_USER", "wallet_user"),
		DBPass:    getEnv("DB_PASSWORD", "secure_password_123"),
		DBName:    getEnv("DB_NAME", "wallet_db"),
		DBSSLMode: getEnv("DB_SSLMODE", "disable"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
