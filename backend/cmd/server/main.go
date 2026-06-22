package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ch-achyuth/auctasy/internal/config"
	"github.com/ch-achyuth/auctasy/internal/database"
	"github.com/ch-achyuth/auctasy/internal/handlers"
	"github.com/ch-achyuth/auctasy/internal/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	log.Println("Connecting to Supabase...")
	db, err := database.Connect(cfg.SupabaseURL, cfg.SupabaseKey)
	if err != nil {
		log.Fatalf("supabase: %v", err)
	}
	log.Println("Connected to Supabase.")

	log.Println("Connecting to Redis...")
	redisDB, err := database.ConnectRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisDB.Close()
	log.Println("Connected to Redis.")

	mux := http.NewServeMux()

	// ── Public endpoints ──────────────────────────────────────────────────────
	mux.HandleFunc("/health", handlers.Health)

	// ── Authenticated endpoints ───────────────────────────────────────────────
	// All /api/v1/* routes require a valid Supabase JWT. The middleware validates
	// the HS256 signature, checks expiry, and injects the user UUID into context.
	auth := middleware.RequireAuth(cfg.JWTSecret)

	mux.Handle("/api/v1/auction/bid", auth(handlers.PlaceBid(db)))

	// ─────────────────────────────────────────────────────────────────────────
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	// Block until SIGINT or SIGTERM, then shut down gracefully.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("forced shutdown: %v", err)
	}
	log.Println("Server stopped.")
}
