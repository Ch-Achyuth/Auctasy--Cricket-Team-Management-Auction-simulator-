package config

import (
	"fmt"
	"os"
	"path/filepath"

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
	// FrontendURL is the allowed WebSocket origin (e.g. http://localhost:8000).
	// Leave empty in dev to allow all origins.
	FrontendURL string
}

// Load reads the .env file and returns a populated Config struct.
//
// The root .env is shared by Go and Vite, but `go run` forces the working
// directory to backend/ (where go.mod lives). So we search the current
// directory and walk up a few parents until a .env is found, letting the
// server start regardless of where it is launched from.
func Load() (*Config, error) {
	loadDotEnv()

	cfg := &Config{
		SupabaseURL: os.Getenv("SUPABASE_URL"),
		SupabaseKey: os.Getenv("SUPABASE_SERVICE_KEY"),
		RedisURL:    os.Getenv("REDIS_URL"),
		JWTSecret:   os.Getenv("SUPABASE_JWT_SECRET"),
		Port:        os.Getenv("PORT"),
		FrontendURL: os.Getenv("FRONTEND_URL"),
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

// loadDotEnv looks for a .env file in the current directory and walks up to
// 4 parent directories, loading the first one it finds. Missing .env is not
// an error — real environments inject vars directly.
func loadDotEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, ".env")
		if _, statErr := os.Stat(candidate); statErr == nil {
			_ = godotenv.Load(candidate)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
}
