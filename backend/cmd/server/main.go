package main

import (
	"fmt"
	"log"

	"github.com/ch-achyuth/auctasy/internal/config"
	"github.com/ch-achyuth/auctasy/internal/database"
)

func main() {
	// Load configuration from .env
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Connect to Supabase via REST API Client
	log.Println("🔌 Connecting to Supabase...")
	db, err := database.Connect(cfg.SupabaseURL, cfg.SupabaseKey)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	
	log.Println("✅ Connected to Supabase successfully!")

	// Connect to Redis
	log.Println("🔌 Connecting to Redis...")
	redisDB, err := database.ConnectRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}
	defer redisDB.Close()
	log.Println("✅ Connected to Redis successfully!")

	// Let's do a simple query to verify: fetch count or 1 item from users table
	var results []map[string]interface{}
	_, _, err = db.Client.From("users").Select("*", "exact", false).Limit(1, "").Execute()
	if err != nil {
		log.Fatalf("❌ Failed to query users table: %v", err)
	}

	fmt.Println("\n📋 Successfully queried the 'users' table via Supabase REST API!")
	fmt.Printf("Fetched %d rows.\n", len(results))
}
