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
	auctionws "github.com/ch-achyuth/auctasy/internal/websocket"
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

	// Hub bridges Redis pub/sub to all connected WebSocket clients.
	// ctx is cancelled on shutdown so hub.Run exits cleanly.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := auctionws.NewHub(redisDB.Client)
	go hub.Run(ctx)

	mux := http.NewServeMux()

	// ── Public endpoints ──────────────────────────────────────────────────────
	mux.HandleFunc("/health", handlers.Health)

	// ── Authenticated REST endpoints ──────────────────────────────────────────
	auth := middleware.RequireAuth(cfg.JWTSecret)

	mux.Handle("/api/v1/auction/bid",
		auth(middleware.BidRateLimit(handlers.PlaceBid(db))))

	mux.Handle("/api/v1/auction/missed-events",
		auth(handlers.MissedEvents(db)))

	// ── WebSocket endpoint ────────────────────────────────────────────────────
	// The browser WebSocket API cannot send custom headers, so the JWT is passed
	// as a query param. We validate it here before upgrading the connection.
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := middleware.ValidateToken(token, cfg.JWTSecret)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		leagueID := r.URL.Query().Get("league_id")
		if leagueID == "" {
			http.Error(w, "league_id required", http.StatusBadRequest)
			return
		}
		auctionws.ServeWS(hub, w, r, userID, leagueID, cfg.FrontendURL)
	})

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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	cancel() // stop the WebSocket hub

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("forced shutdown: %v", err)
	}
	log.Println("Server stopped.")
}
