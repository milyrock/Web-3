package config

import "os"

type Config struct {
	Addr          string
	DatabaseURL   string
	WebDir        string
	AdminEmail    string
	AdminPassword string
}

func Load() Config {
	return Config{
		Addr: value("ADDR", "127.0.0.1:8080"), DatabaseURL: value("DATABASE_URL", "postgres://developer:developer@localhost:5433/developer?sslmode=disable"),
		WebDir:     value("WEB_DIR", "web"),
		AdminEmail: value("ADMIN_EMAIL", "admin@developer.local"), AdminPassword: os.Getenv("ADMIN_PASSWORD"),
	}
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
