package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration values loaded from the environment.
type Config struct {
	SupabaseURL string
	SupabaseKey string
	RedisURL    string
	// JWTSecret is the Supabase JWT secret (Settings → API → JWT Secret).
	// Required for the Go backend to verify user tokens without a round-trip.
	JWTSecret string
	// Port the HTTP server listens on. Defaults to 8080.
	Port string
}

// Load reads the .env file and returns a populated Config struct.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		SupabaseURL: os.Getenv("SUPABASE_URL"),
		SupabaseKey: os.Getenv("SUPABASE_SERVICE_KEY"),
		RedisURL:    os.Getenv("REDIS_URL"),
		JWTSecret:   os.Getenv("SUPABASE_JWT_SECRET"),
		Port:        os.Getenv("PORT"),
	}

	if cfg.SupabaseURL == "" || cfg.SupabaseKey == "" {
		return nil, fmt.Errorf("SUPABASE_URL and SUPABASE_SERVICE_KEY are required")
	}
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("SUPABASE_JWT_SECRET is required")
	}

	return cfg, nil
}
